package pgexcel

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// template_exporter.go - Template-based Excel export engine

// TemplateExporter exports Excel files using YAML template configuration
type TemplateExporter struct {
	db           DB
	template     *ReportTemplate
	templatePath string                 // Path to template file (for resolving relative query files)
	vars         map[string]interface{} // Runtime variables for query parameters
}

// NewTemplateExporter creates a new template-based exporter
func NewTemplateExporter(db DB, template *ReportTemplate) *TemplateExporter {
	return &TemplateExporter{
		db:       db,
		template: template,
		vars:     make(map[string]interface{}),
	}
}

// NewTemplateExporterFromFile creates an exporter by loading a template file
func NewTemplateExporterFromFile(db DB, templatePath string) (*TemplateExporter, error) {
	template, err := LoadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("loading template: %w", err)
	}

	return &TemplateExporter{
		db:           db,
		template:     template,
		templatePath: templatePath,
		vars:         make(map[string]interface{}),
	}, nil
}

// NewTemplateExporterFromString creates an exporter from a YAML string
func NewTemplateExporterFromString(db DB, yamlContent string) (*TemplateExporter, error) {
	template, err := LoadTemplateFromString(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("loading template from string: %w", err)
	}

	return &TemplateExporter{
		db:       db,
		template: template,
		vars:     make(map[string]interface{}),
	}, nil
}

// WithVariables sets runtime variables for query parameters and variable substitution
func (e *TemplateExporter) WithVariables(vars map[string]interface{}) *TemplateExporter {
	for k, v := range vars {
		e.vars[k] = v
	}
	return e
}

// WithVariable sets a single runtime variable
func (e *TemplateExporter) WithVariable(name string, value interface{}) *TemplateExporter {
	e.vars[name] = value
	return e
}

// Export generates Excel file from template and writes to writer
func (e *TemplateExporter) Export(ctx context.Context, writer io.Writer) error {
	// Resolve variables in template
	if err := e.template.ResolveVariables(e.vars); err != nil {
		return fmt.Errorf("resolving variables: %w", err)
	}

	// Create new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Process each sheet
	for i, sheetTmpl := range e.template.Sheets {
		if err := e.exportSheet(ctx, f, &sheetTmpl, i == 0); err != nil {
			return fmt.Errorf("exporting sheet '%s': %w", sheetTmpl.Name, err)
		}
	}

	// Delete default Sheet1 if we created other sheets
	if len(e.template.Sheets) > 0 {
		sheetIndex, _ := f.GetSheetIndex("Sheet1")
		if sheetIndex != -1 && e.template.Sheets[0].Name != "Sheet1" {
			f.DeleteSheet("Sheet1")
		}
	}

	// Write to writer
	if err := f.Write(writer); err != nil {
		return fmt.Errorf("writing Excel file: %w", err)
	}

	return nil
}

// ExportToFile exports to a file path
func (e *TemplateExporter) ExportToFile(ctx context.Context, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	return e.Export(ctx, file)
}

// exportSheet exports a single sheet based on template
func (e *TemplateExporter) exportSheet(ctx context.Context, f *excelize.File, sheetTmpl *SheetTemplate, isFirst bool) error {
	// Get or create sheet
	var sheetIndex int
	var err error

	if isFirst {
		// Rename the default sheet
		f.SetSheetName("Sheet1", sheetTmpl.Name)
		sheetIndex = 0
	} else {
		sheetIndex, err = f.NewSheet(sheetTmpl.Name)
		if err != nil {
			return fmt.Errorf("creating sheet: %w", err)
		}
	}

	// Load query from file if specified
	query := sheetTmpl.Query
	if sheetTmpl.QueryFile != "" {
		basePath := ""
		if e.templatePath != "" {
			basePath = filepath.Dir(e.templatePath)
		}
		query, err = LoadQueryFile(basePath, sheetTmpl.QueryFile)
		if err != nil {
			return err
		}
	}

	// Build query arguments from template references
	queryArgs := e.buildQueryArgs(sheetTmpl.QueryArgs)

	// Execute query
	rows, err := e.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	// Get column info from database
	dbColumns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("getting columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return fmt.Errorf("getting column types: %w", err)
	}

	// Build column map for quick lookup
	columnMap := e.buildColumnMap(sheetTmpl, dbColumns)

	// Create styles
	headerStyle, err := e.createHeaderStyle(f, sheetTmpl)
	if err != nil {
		return fmt.Errorf("creating header style: %w", err)
	}

	dataStyle, err := e.createDataStyle(f, sheetTmpl)
	if err != nil {
		return fmt.Errorf("creating data style: %w", err)
	}

	// Column-specific styles
	colStyles := make(map[int]int)
	for colIdx, dbCol := range dbColumns {
		if tmpl, ok := columnMap[dbCol]; ok && tmpl.Style != nil {
			style, err := e.createStyleFromTemplate(f, tmpl.Style)
			if err != nil {
				return fmt.Errorf("creating column style: %w", err)
			}
			colStyles[colIdx] = style
		}
	}

	rowNum := 1

	// Write headers
	visibleColIdx := 0
	for colIdx, dbCol := range dbColumns {
		tmpl := columnMap[dbCol]

		// Skip hidden columns
		if tmpl != nil && tmpl.Hidden {
			continue
		}

		cell := columnIndexToName(visibleColIdx) + "1"
		header := dbCol
		if tmpl != nil && tmpl.Header != "" {
			header = tmpl.Header
		}

		if err := f.SetCellValue(sheetTmpl.Name, cell, header); err != nil {
			return fmt.Errorf("setting header: %w", err)
		}
		if err := f.SetCellStyle(sheetTmpl.Name, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("setting header style: %w", err)
		}

		// Set column width
		if tmpl != nil && tmpl.Width > 0 {
			if err := f.SetColWidth(sheetTmpl.Name, columnIndexToName(visibleColIdx), columnIndexToName(visibleColIdx), tmpl.Width); err != nil {
				return fmt.Errorf("setting column width: %w", err)
			}
		}

		visibleColIdx++
		_ = colIdx // Used in style lookup
	}
	rowNum++

	// Track column widths for auto-fit
	columnWidths := make([]float64, len(dbColumns))
	for i := range columnWidths {
		columnWidths[i] = 10.0
	}

	// Write data rows
	for rows.Next() {
		values := make([]interface{}, len(dbColumns))
		valuePtrs := make([]interface{}, len(dbColumns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}

		visibleColIdx = 0
		for colIdx, value := range values {
			dbCol := dbColumns[colIdx]
			tmpl := columnMap[dbCol]

			// Skip hidden columns
			if tmpl != nil && tmpl.Hidden {
				continue
			}

			cell := columnIndexToName(visibleColIdx) + fmt.Sprintf("%d", rowNum)

			// Format value
			displayValue := e.formatValue(value, columnTypes[colIdx], tmpl)

			if err := f.SetCellValue(sheetTmpl.Name, cell, displayValue); err != nil {
				return fmt.Errorf("setting cell value: %w", err)
			}

			// Apply style
			styleID := dataStyle
			if s, ok := colStyles[colIdx]; ok {
				styleID = s
			}
			if err := f.SetCellStyle(sheetTmpl.Name, cell, cell, styleID); err != nil {
				return fmt.Errorf("setting cell style: %w", err)
			}

			// Apply conditional formatting
			if tmpl != nil && len(tmpl.Conditional) > 0 {
				e.applyConditionalStyle(f, sheetTmpl.Name, cell, value, tmpl.Conditional)
			}

			// Track width for auto-fit
			if sheetTmpl.Layout != nil && sheetTmpl.Layout.AutoFitCols {
				valueLen := len(fmt.Sprintf("%v", displayValue))
				if float64(valueLen) > columnWidths[visibleColIdx] {
					columnWidths[visibleColIdx] = float64(valueLen)
				}
			}

			visibleColIdx++
		}
		rowNum++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %w", err)
	}

	// Apply layout options
	if err := e.applyLayout(f, sheetTmpl, visibleColIdx, rowNum-1, columnWidths); err != nil {
		return fmt.Errorf("applying layout: %w", err)
	}

	// Apply protection
	if sheetTmpl.Protection != nil && sheetTmpl.Protection.LockSheet {
		if err := e.applyProtection(f, sheetTmpl, visibleColIdx, rowNum-1); err != nil {
			return fmt.Errorf("applying protection: %w", err)
		}
	}

	// Set as active if first sheet
	if isFirst {
		f.SetActiveSheet(sheetIndex)
	}

	return nil
}

// buildColumnMap creates a map from column name to template
func (e *TemplateExporter) buildColumnMap(sheetTmpl *SheetTemplate, dbColumns []string) map[string]*ColumnTemplate {
	colMap := make(map[string]*ColumnTemplate)

	for i := range sheetTmpl.Columns {
		colMap[sheetTmpl.Columns[i].Name] = &sheetTmpl.Columns[i]
	}

	return colMap
}

// buildQueryArgs converts template variable references to actual values
func (e *TemplateExporter) buildQueryArgs(argRefs []string) []interface{} {
	args := make([]interface{}, len(argRefs))
	for i, ref := range argRefs {
		// Check if it's a variable reference
		if strings.HasPrefix(ref, "${") && strings.HasSuffix(ref, "}") {
			varName := ref[2 : len(ref)-1]
			if val, ok := e.vars[varName]; ok {
				args[i] = val
			} else {
				args[i] = nil
			}
		} else {
			args[i] = ref
		}
	}
	return args
}

// formatValue formats a value based on column template
func (e *TemplateExporter) formatValue(value interface{}, colType *sql.ColumnType, tmpl *ColumnTemplate) interface{} {
	if value == nil {
		return ""
	}

	// Handle byte arrays
	if b, ok := value.([]byte); ok {
		return string(b)
	}

	// Handle time values
	if t, ok := value.(time.Time); ok {
		if tmpl != nil && tmpl.Format != "" {
			return t.Format(tmpl.Format)
		}
		return t
	}

	return value
}

// createHeaderStyle creates the header style for a sheet
func (e *TemplateExporter) createHeaderStyle(f *excelize.File, sheetTmpl *SheetTemplate) (int, error) {
	var styleTmpl *StyleTemplate

	// Check sheet-level style first
	if sheetTmpl.Style != nil && sheetTmpl.Style.HeaderStyle != nil {
		styleTmpl = sheetTmpl.Style.HeaderStyle
	} else if e.template.Defaults != nil && e.template.Defaults.HeaderStyle != nil {
		// Fall back to template defaults
		styleTmpl = e.template.Defaults.HeaderStyle
	}

	if styleTmpl != nil {
		return e.createStyleFromTemplate(f, styleTmpl)
	}

	// Use package default
	return e.createStyleFromCellStyle(f, DefaultHeaderStyle())
}

// createDataStyle creates the data style for a sheet
func (e *TemplateExporter) createDataStyle(f *excelize.File, sheetTmpl *SheetTemplate) (int, error) {
	var styleTmpl *StyleTemplate

	if sheetTmpl.Style != nil && sheetTmpl.Style.DataStyle != nil {
		styleTmpl = sheetTmpl.Style.DataStyle
	} else if e.template.Defaults != nil && e.template.Defaults.DataStyle != nil {
		styleTmpl = e.template.Defaults.DataStyle
	}

	if styleTmpl != nil {
		return e.createStyleFromTemplate(f, styleTmpl)
	}

	return e.createStyleFromCellStyle(f, DefaultDataStyle())
}

// createStyleFromTemplate creates an excelize style from StyleTemplate
func (e *TemplateExporter) createStyleFromTemplate(f *excelize.File, tmpl *StyleTemplate) (int, error) {
	cellStyle := tmpl.ToCellStyle()
	if cellStyle == nil {
		return 0, nil
	}
	return e.createStyleFromCellStyle(f, cellStyle)
}

// createStyleFromCellStyle creates an excelize style from CellStyle
func (e *TemplateExporter) createStyleFromCellStyle(f *excelize.File, style *CellStyle) (int, error) {
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

	if style.FontColor != "" {
		excelStyle.Font.Color = strings.TrimPrefix(style.FontColor, "#")
	}

	if style.FillColor != "" {
		excelStyle.Fill = excelize.Fill{
			Type:    "pattern",
			Pattern: style.FillPattern,
			Color:   []string{strings.TrimPrefix(style.FillColor, "#")},
		}
	}

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

	if style.NumberFormat != "" {
		excelStyle.CustomNumFmt = &style.NumberFormat
	}

	return f.NewStyle(excelStyle)
}

// applyLayout applies layout settings from template
func (e *TemplateExporter) applyLayout(f *excelize.File, sheetTmpl *SheetTemplate, numCols, numRows int, columnWidths []float64) error {
	layout := sheetTmpl.Layout
	if layout == nil {
		return nil
	}

	// Freeze panes
	if layout.FreezeRows > 0 || layout.FreezeCols > 0 {
		topLeftCell := columnIndexToName(layout.FreezeCols) + fmt.Sprintf("%d", layout.FreezeRows+1)
		if err := f.SetPanes(sheetTmpl.Name, &excelize.Panes{
			Freeze:      true,
			XSplit:      layout.FreezeCols,
			YSplit:      layout.FreezeRows,
			TopLeftCell: topLeftCell,
			ActivePane:  "bottomRight",
		}); err != nil {
			return fmt.Errorf("setting freeze panes: %w", err)
		}
	}

	// Auto filter
	if layout.AutoFilter && numCols > 0 {
		lastCol := columnIndexToName(numCols - 1)
		filterRange := fmt.Sprintf("A1:%s1", lastCol)
		if err := f.AutoFilter(sheetTmpl.Name, filterRange, []excelize.AutoFilterOptions{}); err != nil {
			return fmt.Errorf("setting auto filter: %w", err)
		}
	}

	// Auto-fit columns
	if layout.AutoFitCols {
		maxWidth := float64(50)
		if layout.MaxColWidth > 0 {
			maxWidth = float64(layout.MaxColWidth)
		}

		for i, width := range columnWidths {
			if i >= numCols {
				break
			}
			colName := columnIndexToName(i)
			adjustedWidth := width * 1.2
			if adjustedWidth > maxWidth {
				adjustedWidth = maxWidth
			}
			if err := f.SetColWidth(sheetTmpl.Name, colName, colName, adjustedWidth); err != nil {
				return fmt.Errorf("setting column width: %w", err)
			}
		}
	}

	return nil
}

// applyProtection applies protection settings from template
func (e *TemplateExporter) applyProtection(f *excelize.File, sheetTmpl *SheetTemplate, numCols, numRows int) error {
	protection := sheetTmpl.Protection
	if protection == nil {
		return nil
	}

	sp := protection.ToSheetProtection()
	if sp == nil {
		return nil
	}

	// Unlock specified columns
	if len(protection.UnlockedColumns) > 0 {
		unlockedStyle, err := f.NewStyle(&excelize.Style{
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return fmt.Errorf("creating unlocked style: %w", err)
		}

		for _, col := range protection.UnlockedColumns {
			// Find column index by searching for matching column name
			colIdx := e.findColumnIndex(sheetTmpl, col)
			if colIdx >= 0 {
				colName := columnIndexToName(colIdx)
				// Apply to data rows (skip header)
				startCell := colName + "2"
				endCell := colName + fmt.Sprintf("%d", numRows)
				if err := f.SetCellStyle(sheetTmpl.Name, startCell, endCell, unlockedStyle); err != nil {
					return fmt.Errorf("unlocking column %s: %w", col, err)
				}
			}
		}
	}

	// Apply unlocked ranges
	if len(sp.UnlockedRanges) > 0 {
		unlockedStyle, err := f.NewStyle(&excelize.Style{
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return fmt.Errorf("creating unlocked style: %w", err)
		}

		for _, rng := range sp.UnlockedRanges {
			startCell := fmt.Sprintf("%s%d", rng.StartCol, rng.StartRow)
			endCell := fmt.Sprintf("%s%d", rng.EndCol, rng.EndRow)
			if err := f.SetCellStyle(sheetTmpl.Name, startCell, endCell, unlockedStyle); err != nil {
				return fmt.Errorf("unlocking range: %w", err)
			}
		}
	}

	// Enable sheet protection
	protectOpts := &excelize.SheetProtectionOptions{
		Password:         sp.Password,
		FormatCells:      sp.AllowFormatCells,
		FormatColumns:    sp.AllowFormatColumns,
		FormatRows:       sp.AllowFormatRows,
		InsertColumns:    sp.AllowInsertColumns,
		InsertRows:       sp.AllowInsertRows,
		InsertHyperlinks: sp.AllowInsertHyperlinks,
		DeleteColumns:    sp.AllowDeleteColumns,
		DeleteRows:       sp.AllowDeleteRows,
		Sort:             sp.AllowSort,
		AutoFilter:       sp.AllowFilter,
		PivotTables:      sp.AllowPivotTables,
	}

	if err := f.ProtectSheet(sheetTmpl.Name, protectOpts); err != nil {
		return fmt.Errorf("protecting sheet: %w", err)
	}

	return nil
}

// findColumnIndex finds the visible column index for a column name
func (e *TemplateExporter) findColumnIndex(sheetTmpl *SheetTemplate, colName string) int {
	idx := 0
	for _, col := range sheetTmpl.Columns {
		if col.Hidden {
			continue
		}
		if col.Name == colName {
			return idx
		}
		idx++
	}
	return -1
}

// applyConditionalStyle applies conditional formatting based on rules
func (e *TemplateExporter) applyConditionalStyle(f *excelize.File, sheetName, cell string, value interface{}, rules []ConditionalRule) {
	for _, rule := range rules {
		if e.evaluateCondition(value, rule.Condition) && rule.Style != nil {
			style, err := e.createStyleFromTemplate(f, rule.Style)
			if err == nil && style != 0 {
				f.SetCellStyle(sheetName, cell, cell, style)
			}
			break // Apply first matching rule
		}
	}
}

// evaluateCondition evaluates a simple condition expression
func (e *TemplateExporter) evaluateCondition(value interface{}, condition string) bool {
	if value == nil || condition == "" {
		return false
	}

	condition = strings.TrimSpace(condition)

	// Parse condition: operator + value
	// Supported: >, <, >=, <=, ==, !=, contains

	// Handle contains
	if strings.HasPrefix(condition, "contains ") {
		searchStr := strings.TrimPrefix(condition, "contains ")
		searchStr = strings.Trim(searchStr, "'\"")
		return strings.Contains(fmt.Sprintf("%v", value), searchStr)
	}

	// Handle comparison operators
	operators := []string{">=", "<=", "!=", "==", ">", "<"}
	for _, op := range operators {
		if strings.HasPrefix(condition, op) {
			compareVal := strings.TrimSpace(strings.TrimPrefix(condition, op))
			return e.compareValues(value, op, compareVal)
		}
	}

	return false
}

// compareValues compares a value against a condition value
func (e *TemplateExporter) compareValues(value interface{}, operator, compareStr string) bool {
	// Handle string comparison
	compareStr = strings.Trim(compareStr, "'\"")

	// Try numeric comparison first
	switch v := value.(type) {
	case int, int32, int64, float32, float64:
		floatVal := toFloat64(v)
		compareFloat, err := strconv.ParseFloat(compareStr, 64)
		if err == nil {
			switch operator {
			case ">":
				return floatVal > compareFloat
			case "<":
				return floatVal < compareFloat
			case ">=":
				return floatVal >= compareFloat
			case "<=":
				return floatVal <= compareFloat
			case "==":
				return floatVal == compareFloat
			case "!=":
				return floatVal != compareFloat
			}
		}
	case string:
		switch operator {
		case "==":
			return v == compareStr
		case "!=":
			return v != compareStr
		case ">":
			return v > compareStr
		case "<":
			return v < compareStr
		case ">=":
			return v >= compareStr
		case "<=":
			return v <= compareStr
		}
	}

	// Default string comparison
	strVal := fmt.Sprintf("%v", value)
	switch operator {
	case "==":
		return strVal == compareStr
	case "!=":
		return strVal != compareStr
	}

	return false
}

// toFloat64 converts numeric types to float64
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

// Compile-time check for regex patterns
var cellRangePattern = regexp.MustCompile(`^[A-Z]+\d+:[A-Z]+\d+$`)
