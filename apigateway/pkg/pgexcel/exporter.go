package pgexcel

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// PgExcelExporter implements the Exporter interface
type PgExcelExporter struct {
	db     DB
	config *ExportConfig
}

// NewExporter creates a new PostgreSQL to Excel exporter
func NewExporter(db DB) *PgExcelExporter {
	return &PgExcelExporter{
		db: db,
		config: &ExportConfig{
			IncludeHeaders: true,
			SheetName:      "Sheet1",
			FreezeHeader:   false,
			AutoFilter:     false,
			AutoFitColumns: true,
			MaxColumnWidth: 50,
			DateFormat:     "2006-01-02",
			TimeFormat:     "15:04:05",
			NumberFormat:   "#,##0.00",
			HeaderStyle:    DefaultHeaderStyle(),
			DataStyles:     make(map[string]*CellStyle),
			Sheets:         []SheetConfig{},
		},
	}
}

// WithQuery sets the SQL query to execute
func (e *PgExcelExporter) WithQuery(query string, args ...interface{}) *PgExcelExporter {
	e.config.Query = query
	e.config.Args = args
	return e
}

// WithSheetName sets the sheet name
func (e *PgExcelExporter) WithSheetName(name string) *PgExcelExporter {
	e.config.SheetName = name
	return e
}

// WithHeaders enables or disables header row
func (e *PgExcelExporter) WithHeaders(include bool) *PgExcelExporter {
	e.config.IncludeHeaders = include
	return e
}

// WithProtection sets sheet protection
func (e *PgExcelExporter) WithProtection(rules ...ProtectionRule) *PgExcelExporter {
	sp := NewSheetProtection()
	for _, rule := range rules {
		rule.Apply(sp)
	}
	e.config.Protection = sp
	return e
}

// WithPassword sets protection password
func (e *PgExcelExporter) WithPassword(password string) *PgExcelExporter {
	if e.config.Protection == nil {
		e.config.Protection = NewSheetProtection()
	}
	e.config.Protection.Password = password
	return e
}

// AddSheet adds another sheet to the workbook
func (e *PgExcelExporter) AddSheet(query string, sheetName string, opts ...SheetOption) Exporter {
	sheetCfg := SheetConfig{
		Query:     query,
		SheetName: sheetName,
		Options:   opts,
	}

	// Apply options
	for _, opt := range opts {
		opt(&sheetCfg)
	}

	e.config.Sheets = append(e.config.Sheets, sheetCfg)
	return e
}

// Export executes the query and exports to Excel
func (e *PgExcelExporter) Export(ctx context.Context, writer io.Writer, opts ...ExportOption) error {
	// Apply export options
	for _, opt := range opts {
		if err := opt(e.config); err != nil {
			return fmt.Errorf("applying export option: %w", err)
		}
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Export main query if specified
	if e.config.Query != "" {
		if err := e.exportSheet(ctx, f, e.config.SheetName, e.config.Query, e.config.Args, e.config); err != nil {
			return fmt.Errorf("exporting main sheet: %w", err)
		}

		// Set as active sheet
		f.SetActiveSheet(0)
	}

	// Export additional sheets
	for i, sheetCfg := range e.config.Sheets {
		sheetIndex, err := f.NewSheet(sheetCfg.SheetName)
		if err != nil {
			return fmt.Errorf("creating sheet %s: %w", sheetCfg.SheetName, err)
		}

		// Create a config for this sheet
		sheetExportCfg := &ExportConfig{
			Query:          sheetCfg.Query,
			Args:           sheetCfg.Args,
			SheetName:      sheetCfg.SheetName,
			IncludeHeaders: e.config.IncludeHeaders,
			FreezeHeader:   e.config.FreezeHeader,
			AutoFilter:     e.config.AutoFilter,
			AutoFitColumns: e.config.AutoFitColumns,
			MaxColumnWidth: e.config.MaxColumnWidth,
			Protection:     sheetCfg.Protection,
			HeaderStyle:    e.config.HeaderStyle,
			DataStyles:     e.config.DataStyles,
			DateFormat:     e.config.DateFormat,
			TimeFormat:     e.config.TimeFormat,
			NumberFormat:   e.config.NumberFormat,
		}

		if err := e.exportSheet(ctx, f, sheetCfg.SheetName, sheetCfg.Query, sheetCfg.Args, sheetExportCfg); err != nil {
			return fmt.Errorf("exporting sheet %s: %w", sheetCfg.SheetName, err)
		}

		if i == 0 && e.config.Query == "" {
			f.SetActiveSheet(sheetIndex)
		}
	}

	// Delete default Sheet1 if we didn't use it
	if e.config.Query == "" && len(e.config.Sheets) > 0 {
		f.DeleteSheet("Sheet1")
	}

	// Write to writer
	if err := f.Write(writer); err != nil {
		return fmt.Errorf("writing Excel file: %w", err)
	}

	return nil
}

// ExportToFile exports to a file path
func (e *PgExcelExporter) ExportToFile(ctx context.Context, filepath string, opts ...ExportOption) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	return e.Export(ctx, file, opts...)
}

// exportSheet exports a single sheet
func (e *PgExcelExporter) exportSheet(ctx context.Context, f *excelize.File, sheetName, query string, args []interface{}, cfg *ExportConfig) error {
	// Execute query
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("getting columns: %w", err)
	}

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("getting column types: %w", err)
	}

	// Create or get sheet
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if sheetIndex == -1 {
		sheetIndex, err = f.NewSheet(sheetName)
		if err != nil {
			return fmt.Errorf("creating sheet: %w", err)
		}
	}

	// Create styles
	headerStyleID, err := e.createStyle(f, cfg.HeaderStyle)
	if err != nil {
		return fmt.Errorf("creating header style: %w", err)
	}

	dataStyleID, err := e.createStyle(f, DefaultDataStyle())
	if err != nil {
		return fmt.Errorf("creating data style: %w", err)
	}

	rowNum := 1

	// Write headers
	if cfg.IncludeHeaders {
		for colIdx, colName := range columns {
			cell := columnIndexToName(colIdx) + "1"
			if err := f.SetCellValue(sheetName, cell, colName); err != nil {
				return fmt.Errorf("setting header value: %w", err)
			}
			if err := f.SetCellStyle(sheetName, cell, cell, headerStyleID); err != nil {
				return fmt.Errorf("setting header style: %w", err)
			}
		}
		rowNum++
	}

	// Prepare column widths tracking
	columnWidths := make([]float64, len(columns))
	for i := range columnWidths {
		columnWidths[i] = 10.0 // Default width
	}

	// Write data rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}

		for colIdx, value := range values {
			cell := columnIndexToName(colIdx) + fmt.Sprintf("%d", rowNum)

			// Convert value based on type
			displayValue := e.formatValue(value, columnTypes[colIdx], cfg)

			if err := f.SetCellValue(sheetName, cell, displayValue); err != nil {
				return fmt.Errorf("setting cell value: %w", err)
			}

			// Apply style
			if err := f.SetCellStyle(sheetName, cell, cell, dataStyleID); err != nil {
				return fmt.Errorf("setting cell style: %w", err)
			}

			// Track column width
			if cfg.AutoFitColumns {
				valueLen := len(fmt.Sprintf("%v", displayValue))
				if float64(valueLen) > columnWidths[colIdx] {
					columnWidths[colIdx] = float64(valueLen)
				}
			}
		}

		rowNum++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %w", err)
	}

	// Set column widths
	if cfg.AutoFitColumns {
		for i, width := range columnWidths {
			colName := columnIndexToName(i)
			adjustedWidth := width * 1.2 // Add some padding
			if adjustedWidth > float64(cfg.MaxColumnWidth) {
				adjustedWidth = float64(cfg.MaxColumnWidth)
			}
			if err := f.SetColWidth(sheetName, colName, colName, adjustedWidth); err != nil {
				return fmt.Errorf("setting column width: %w", err)
			}
		}
	}

	// Apply freeze panes
	if cfg.FreezeHeader && cfg.IncludeHeaders {
		if err := f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			XSplit:      0,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		}); err != nil {
			return fmt.Errorf("setting freeze panes: %w", err)
		}
	}

	// Apply auto filter
	if cfg.AutoFilter && cfg.IncludeHeaders {
		lastCol := columnIndexToName(len(columns) - 1)
		filterRange := fmt.Sprintf("A1:%s1", lastCol)
		if err := f.AutoFilter(sheetName, filterRange, []excelize.AutoFilterOptions{}); err != nil {
			return fmt.Errorf("setting auto filter: %w", err)
		}
	}

	// Apply protection
	if cfg.Protection != nil && cfg.Protection.ProtectSheet {
		if err := e.applyProtection(f, sheetName, cfg.Protection, len(columns), rowNum-1); err != nil {
			return fmt.Errorf("applying protection: %w", err)
		}
	}

	return nil
}

// createStyle creates an Excel style from a CellStyle
func (e *PgExcelExporter) createStyle(f *excelize.File, style *CellStyle) (int, error) {
	if style == nil {
		return 0, nil
	}

	excelStyle := &excelize.Style{
		Font: &excelize.Font{
			Bold:   style.FontBold,
			Italic: style.FontItalic,
			Size:   style.FontSize,
			Family: style.FontName,
		},
		Alignment: &excelize.Alignment{
			Horizontal: style.Alignment,
			Vertical:   style.VerticalAlign,
			WrapText:   style.WrapText,
		},
		Protection: &excelize.Protection{
			Locked: style.Locked,
		},
	}

	// Set font color
	if style.FontColor != "" {
		excelStyle.Font.Color = strings.TrimPrefix(style.FontColor, "#")
	}

	// Set fill
	if style.FillColor != "" {
		excelStyle.Fill = excelize.Fill{
			Type:    "pattern",
			Pattern: style.FillPattern,
			Color:   []string{strings.TrimPrefix(style.FillColor, "#")},
		}
	}

	// Set border
	if style.BorderStyle != "" {
		borderColor := "000000"
		if style.BorderColor != "" {
			borderColor = strings.TrimPrefix(style.BorderColor, "#")
		}
		excelStyle.Border = []excelize.Border{
			{Type: "left", Color: borderColor, Style: 1},
			{Type: "top", Color: borderColor, Style: 1},
			{Type: "bottom", Color: borderColor, Style: 1},
			{Type: "right", Color: borderColor, Style: 1},
		}
	}

	// Set number format
	if style.NumberFormat != "" {
		excelStyle.CustomNumFmt = &style.NumberFormat
	}

	return f.NewStyle(excelStyle)
}

// formatValue formats a database value for Excel
func (e *PgExcelExporter) formatValue(value interface{}, colType *sql.ColumnType, cfg *ExportConfig) interface{} {
	if value == nil {
		return ""
	}

	// Handle byte arrays (common for strings in PostgreSQL)
	if b, ok := value.([]byte); ok {
		return string(b)
	}

	// Handle time values
	if t, ok := value.(time.Time); ok {
		return t
	}

	return value
}

// applyProtection applies protection rules to the sheet
func (e *PgExcelExporter) applyProtection(f *excelize.File, sheetName string, protection *SheetProtection, numCols, numRows int) error {
	// First, unlock all cells by default if there are unlock ranges
	if len(protection.UnlockedRanges) > 0 {
		// Create an unlocked style
		unlockedStyle, err := f.NewStyle(&excelize.Style{
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return fmt.Errorf("creating unlocked style: %w", err)
		}

		// Apply to unlocked ranges
		for _, rng := range protection.UnlockedRanges {
			rangeStr := fmt.Sprintf("%s%d:%s%d", rng.StartCol, rng.StartRow, rng.EndCol, rng.EndRow)
			if err := f.SetCellStyle(sheetName, fmt.Sprintf("%s%d", rng.StartCol, rng.StartRow),
				fmt.Sprintf("%s%d", rng.EndCol, rng.EndRow), unlockedStyle); err != nil {
				return fmt.Errorf("unlocking range %s: %w", rangeStr, err)
			}
		}
	}

	// Enable sheet protection using correct API
	enableProtection := &excelize.SheetProtectionOptions{
		Password:         protection.Password,
		EditScenarios:    protection.AllowFormatCells,
		FormatCells:      protection.AllowFormatCells,
		FormatColumns:    protection.AllowFormatColumns,
		FormatRows:       protection.AllowFormatRows,
		InsertColumns:    protection.AllowInsertColumns,
		InsertRows:       protection.AllowInsertRows,
		InsertHyperlinks: protection.AllowInsertHyperlinks,
		DeleteColumns:    protection.AllowDeleteColumns,
		DeleteRows:       protection.AllowDeleteRows,
		Sort:             protection.AllowSort,
		AutoFilter:       protection.AllowFilter,
		PivotTables:      protection.AllowPivotTables,
	}

	if err := f.ProtectSheet(sheetName, enableProtection); err != nil {
		return fmt.Errorf("protecting sheet: %w", err)
	}

	return nil
}

// columnIndexToName converts column index (0-based) to Excel column name
func columnIndexToName(index int) string {
	name := ""
	index++ // Convert to 1-based

	for index > 0 {
		index--
		name = string(rune('A'+index%26)) + name
		index /= 26
	}

	return name
}
