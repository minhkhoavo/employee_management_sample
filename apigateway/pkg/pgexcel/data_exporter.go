package pgexcel

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// data_exporter.go - Export Go data structures (slices, structs, maps) to Excel

// DataExporter exports in-memory Go data to Excel using templates
type DataExporter struct {
	template *ReportTemplate
	data     map[string]interface{} // Sheet name -> data slice
}

// NewDataExporter creates a new data exporter with optional template
func NewDataExporter() *DataExporter {
	return &DataExporter{
		data: make(map[string]interface{}),
	}
}

// NewDataExporterWithTemplate creates a data exporter with a YAML template
func NewDataExporterWithTemplate(template *ReportTemplate) *DataExporter {
	return &DataExporter{
		template: template,
		data:     make(map[string]interface{}),
	}
}

// NewDataExporterFromTemplateFile creates a data exporter from a YAML template file
func NewDataExporterFromTemplateFile(templatePath string) (*DataExporter, error) {
	template, err := LoadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("loading template: %w", err)
	}
	return NewDataExporterWithTemplate(template), nil
}

// WithData adds data for a sheet (data should be a slice of structs or maps)
func (e *DataExporter) WithData(sheetName string, data interface{}) *DataExporter {
	e.data[sheetName] = data
	return e
}

// AddSheet adds a sheet with data using a fluent builder pattern
func (e *DataExporter) AddSheet(sheetName string) *SheetBuilder {
	return &SheetBuilder{
		exporter:  e,
		sheetName: sheetName,
	}
}

// Export writes the Excel file to the provided writer
func (e *DataExporter) Export(ctx context.Context, writer io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetIdx := 0
	for sheetName, data := range e.data {
		var sheetTmpl *SheetTemplate
		if e.template != nil {
			for i := range e.template.Sheets {
				if e.template.Sheets[i].Name == sheetName {
					sheetTmpl = &e.template.Sheets[i]
					break
				}
			}
		}

		if err := e.exportSheet(f, sheetName, data, sheetTmpl, sheetIdx == 0); err != nil {
			return fmt.Errorf("exporting sheet '%s': %w", sheetName, err)
		}
		sheetIdx++
	}

	// Delete default Sheet1 if we created other sheets and didn't use it
	if len(e.data) > 0 {
		for sheetName := range e.data {
			if sheetName != "Sheet1" {
				f.DeleteSheet("Sheet1")
				break
			}
		}
	}

	return f.Write(writer)
}

// ExportToFile exports to a file path
func (e *DataExporter) ExportToFile(ctx context.Context, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()
	return e.Export(ctx, file)
}

// exportSheet exports a single sheet from data
func (e *DataExporter) exportSheet(f *excelize.File, sheetName string, data interface{}, tmpl *SheetTemplate, isFirst bool) error {
	// Create or rename sheet
	if isFirst {
		f.SetSheetName("Sheet1", sheetName)
	} else {
		if _, err := f.NewSheet(sheetName); err != nil {
			return fmt.Errorf("creating sheet: %w", err)
		}
	}

	// Get data as slice using reflection
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Ptr {
		dataVal = dataVal.Elem()
	}
	if dataVal.Kind() != reflect.Slice {
		return fmt.Errorf("data must be a slice, got %s", dataVal.Kind())
	}

	if dataVal.Len() == 0 {
		return nil // Empty data, nothing to export
	}

	// Extract column info from first element
	columns, err := e.extractColumns(dataVal.Index(0), tmpl)
	if err != nil {
		return fmt.Errorf("extracting columns: %w", err)
	}

	// Create styles
	headerStyle, dataStyle, colStyles, err := e.createStyles(f, tmpl, columns)
	if err != nil {
		return fmt.Errorf("creating styles: %w", err)
	}

	// Write headers
	for colIdx, col := range columns {
		cell := columnIndexToName(colIdx) + "1"
		if err := f.SetCellValue(sheetName, cell, col.Header); err != nil {
			return fmt.Errorf("setting header: %w", err)
		}
		if err := f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("setting header style: %w", err)
		}
		if col.Width > 0 {
			colName := columnIndexToName(colIdx)
			if err := f.SetColWidth(sheetName, colName, colName, col.Width); err != nil {
				return fmt.Errorf("setting column width: %w", err)
			}
		}
	}

	// Write data rows
	for rowIdx := 0; rowIdx < dataVal.Len(); rowIdx++ {
		rowVal := dataVal.Index(rowIdx)
		rowNum := rowIdx + 2 // 1-based, skip header

		for colIdx, col := range columns {
			cell := columnIndexToName(colIdx) + fmt.Sprintf("%d", rowNum)
			value := e.getFieldValue(rowVal, col.FieldName)
			displayValue := e.formatDataValue(value, col)

			if err := f.SetCellValue(sheetName, cell, displayValue); err != nil {
				return fmt.Errorf("setting cell value: %w", err)
			}

			// Apply style
			styleID := dataStyle
			if s, ok := colStyles[colIdx]; ok {
				styleID = s
			}
			if err := f.SetCellStyle(sheetName, cell, cell, styleID); err != nil {
				return fmt.Errorf("setting cell style: %w", err)
			}

			// Apply conditional formatting
			if col.Conditional != nil && len(col.Conditional) > 0 {
				e.applyConditionalStyle(f, sheetName, cell, value, col.Conditional)
			}
		}
	}

	numRows := dataVal.Len() + 1 // Include header

	// Apply layout from template
	if tmpl != nil && tmpl.Layout != nil {
		if err := e.applyLayout(f, sheetName, len(columns), numRows, tmpl.Layout); err != nil {
			return fmt.Errorf("applying layout: %w", err)
		}
	}

	// Apply protection from template
	if tmpl != nil && tmpl.Protection != nil && tmpl.Protection.LockSheet {
		if err := e.applyProtection(f, sheetName, columns, numRows, tmpl.Protection); err != nil {
			return fmt.Errorf("applying protection: %w", err)
		}
	}

	return nil
}

// ColumnInfo holds extracted column information
type ColumnInfo struct {
	FieldName   string
	Header      string
	Width       float64
	Format      string
	Hidden      bool
	Conditional []ConditionalRule
	Style       *StyleTemplate
}

// extractColumns extracts column information from a struct/map
func (e *DataExporter) extractColumns(val reflect.Value, tmpl *SheetTemplate) ([]ColumnInfo, error) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var columns []ColumnInfo

	switch val.Kind() {
	case reflect.Struct:
		columns = e.extractColumnsFromStruct(val, tmpl)
	case reflect.Map:
		columns = e.extractColumnsFromMap(val, tmpl)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", val.Kind())
	}

	// Filter out hidden columns
	var visible []ColumnInfo
	for _, col := range columns {
		if !col.Hidden {
			visible = append(visible, col)
		}
	}

	return visible, nil
}

// extractColumnsFromStruct extracts columns from a struct type
func (e *DataExporter) extractColumnsFromStruct(val reflect.Value, tmpl *SheetTemplate) []ColumnInfo {
	valType := val.Type()
	var columns []ColumnInfo

	for i := 0; i < valType.NumField(); i++ {
		field := valType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		col := ColumnInfo{
			FieldName: field.Name,
			Header:    field.Name,
		}

		// Check for excel tag: `excel:"header:Name,width:20,format:$#,##0.00"`
		if tag := field.Tag.Get("excel"); tag != "" {
			e.parseExcelTag(&col, tag)
		}

		// Check for json tag as fallback for header
		if col.Header == field.Name {
			if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					col.Header = parts[0]
				}
			}
		}

		// Override with template settings if available
		if tmpl != nil {
			if colTmpl := tmpl.GetColumnByName(field.Name); colTmpl != nil {
				if colTmpl.Header != "" {
					col.Header = colTmpl.Header
				}
				if colTmpl.Width > 0 {
					col.Width = colTmpl.Width
				}
				if colTmpl.Format != "" {
					col.Format = colTmpl.Format
				}
				col.Hidden = colTmpl.Hidden
				col.Conditional = colTmpl.Conditional
				col.Style = colTmpl.Style
			}
		}

		columns = append(columns, col)
	}

	return columns
}

// extractColumnsFromMap extracts columns from a map
func (e *DataExporter) extractColumnsFromMap(val reflect.Value, tmpl *SheetTemplate) []ColumnInfo {
	var columns []ColumnInfo

	for _, key := range val.MapKeys() {
		keyStr := fmt.Sprintf("%v", key.Interface())
		col := ColumnInfo{
			FieldName: keyStr,
			Header:    keyStr,
		}

		// Override with template settings
		if tmpl != nil {
			if colTmpl := tmpl.GetColumnByName(keyStr); colTmpl != nil {
				if colTmpl.Header != "" {
					col.Header = colTmpl.Header
				}
				if colTmpl.Width > 0 {
					col.Width = colTmpl.Width
				}
				if colTmpl.Format != "" {
					col.Format = colTmpl.Format
				}
				col.Hidden = colTmpl.Hidden
				col.Conditional = colTmpl.Conditional
				col.Style = colTmpl.Style
			}
		}

		columns = append(columns, col)
	}

	return columns
}

// parseExcelTag parses the excel struct tag
func (e *DataExporter) parseExcelTag(col *ColumnInfo, tag string) {
	if tag == "-" {
		col.Hidden = true
		return
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		switch key {
		case "header":
			col.Header = value
		case "width":
			fmt.Sscanf(value, "%f", &col.Width)
		case "format":
			col.Format = value
		case "hidden":
			col.Hidden = value == "true"
		}
	}
}

// getFieldValue gets a field value from a struct or map
func (e *DataExporter) getFieldValue(val reflect.Value, fieldName string) interface{} {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			return nil
		}
		return field.Interface()
	case reflect.Map:
		mapVal := val.MapIndex(reflect.ValueOf(fieldName))
		if !mapVal.IsValid() {
			return nil
		}
		return mapVal.Interface()
	}

	return nil
}

// formatDataValue formats a value for Excel
func (e *DataExporter) formatDataValue(value interface{}, col ColumnInfo) interface{} {
	if value == nil {
		return ""
	}

	// Handle time.Time
	if t, ok := value.(time.Time); ok {
		if col.Format != "" {
			return t.Format(col.Format)
		}
		return t
	}

	// Handle pointer types
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		return e.formatDataValue(val.Elem().Interface(), col)
	}

	return value
}

// createStyles creates Excel styles
func (e *DataExporter) createStyles(f *excelize.File, tmpl *SheetTemplate, columns []ColumnInfo) (int, int, map[int]int, error) {
	var headerStyleTmpl, dataStyleTmpl *StyleTemplate

	if tmpl != nil && tmpl.Style != nil {
		headerStyleTmpl = tmpl.Style.HeaderStyle
		dataStyleTmpl = tmpl.Style.DataStyle
	}

	// Default header style
	headerStyle := 0
	if headerStyleTmpl != nil {
		s, err := e.createStyleFromTemplate(f, headerStyleTmpl)
		if err != nil {
			return 0, 0, nil, err
		}
		headerStyle = s
	} else {
		s, err := e.createStyleFromCellStyle(f, DefaultHeaderStyle())
		if err != nil {
			return 0, 0, nil, err
		}
		headerStyle = s
	}

	// Default data style
	dataStyle := 0
	if dataStyleTmpl != nil {
		s, err := e.createStyleFromTemplate(f, dataStyleTmpl)
		if err != nil {
			return 0, 0, nil, err
		}
		dataStyle = s
	} else {
		s, err := e.createStyleFromCellStyle(f, DefaultDataStyle())
		if err != nil {
			return 0, 0, nil, err
		}
		dataStyle = s
	}

	// Column-specific styles
	colStyles := make(map[int]int)
	for i, col := range columns {
		if col.Style != nil {
			s, err := e.createStyleFromTemplate(f, col.Style)
			if err != nil {
				return 0, 0, nil, err
			}
			colStyles[i] = s
		}
	}

	return headerStyle, dataStyle, colStyles, nil
}

// createStyleFromTemplate creates an excelize style from StyleTemplate
func (e *DataExporter) createStyleFromTemplate(f *excelize.File, tmpl *StyleTemplate) (int, error) {
	if tmpl == nil {
		return 0, nil
	}
	cellStyle := tmpl.ToCellStyle()
	return e.createStyleFromCellStyle(f, cellStyle)
}

// createStyleFromCellStyle creates an excelize style from CellStyle
func (e *DataExporter) createStyleFromCellStyle(f *excelize.File, style *CellStyle) (int, error) {
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

	if style.NumberFormat != "" {
		excelStyle.CustomNumFmt = &style.NumberFormat
	}

	return f.NewStyle(excelStyle)
}

// applyLayout applies layout settings
func (e *DataExporter) applyLayout(f *excelize.File, sheetName string, numCols, numRows int, layout *LayoutTemplate) error {
	// Freeze panes
	if layout.FreezeRows > 0 || layout.FreezeCols > 0 {
		topLeftCell := columnIndexToName(layout.FreezeCols) + fmt.Sprintf("%d", layout.FreezeRows+1)
		if err := f.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			XSplit:      layout.FreezeCols,
			YSplit:      layout.FreezeRows,
			TopLeftCell: topLeftCell,
			ActivePane:  "bottomRight",
		}); err != nil {
			return err
		}
	}

	// Auto filter
	if layout.AutoFilter && numCols > 0 {
		lastCol := columnIndexToName(numCols - 1)
		filterRange := fmt.Sprintf("A1:%s1", lastCol)
		if err := f.AutoFilter(sheetName, filterRange, []excelize.AutoFilterOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// applyProtection applies protection settings
func (e *DataExporter) applyProtection(f *excelize.File, sheetName string, columns []ColumnInfo, numRows int, protection *ProtectionTemplate) error {
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
			return err
		}

		for _, colName := range protection.UnlockedColumns {
			// Find column index
			colIdx := -1
			for i, col := range columns {
				if col.FieldName == colName || col.Header == colName {
					colIdx = i
					break
				}
			}
			if colIdx >= 0 {
				colLetter := columnIndexToName(colIdx)
				startCell := colLetter + "2"
				endCell := colLetter + fmt.Sprintf("%d", numRows)
				if err := f.SetCellStyle(sheetName, startCell, endCell, unlockedStyle); err != nil {
					return err
				}
			}
		}
	}

	// Enable sheet protection
	protectOpts := &excelize.SheetProtectionOptions{
		Password:   sp.Password,
		AutoFilter: sp.AllowFilter,
		Sort:       sp.AllowSort,
	}

	return f.ProtectSheet(sheetName, protectOpts)
}

// applyConditionalStyle applies conditional formatting
func (e *DataExporter) applyConditionalStyle(f *excelize.File, sheetName, cell string, value interface{}, rules []ConditionalRule) {
	for _, rule := range rules {
		if evaluateCondition(value, rule.Condition) && rule.Style != nil {
			style, err := e.createStyleFromTemplate(f, rule.Style)
			if err == nil && style != 0 {
				f.SetCellStyle(sheetName, cell, cell, style)
			}
			break
		}
	}
}

// evaluateCondition evaluates a condition (reuse from template_exporter)
func evaluateCondition(value interface{}, condition string) bool {
	if value == nil || condition == "" {
		return false
	}

	condition = strings.TrimSpace(condition)

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
			return compareValues(value, op, compareVal)
		}
	}

	return false
}

// compareValues compares values (helper)
func compareValues(value interface{}, operator, compareStr string) bool {
	compareStr = strings.Trim(compareStr, "'\"")

	switch v := value.(type) {
	case int, int32, int64, float32, float64:
		floatVal := toFloat64(v)
		var compareFloat float64
		fmt.Sscanf(compareStr, "%f", &compareFloat)

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
	case string:
		switch operator {
		case "==":
			return v == compareStr
		case "!=":
			return v != compareStr
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

// SheetBuilder provides a fluent API for building sheets
type SheetBuilder struct {
	exporter   *DataExporter
	sheetName  string
	sheetData  interface{}
	columns    []ColumnInfo
	layout     *LayoutTemplate
	protection *ProtectionTemplate
}

// WithData sets the data for this sheet
func (b *SheetBuilder) WithData(data interface{}) *SheetBuilder {
	b.sheetData = data
	return b
}

// WithColumns sets custom column definitions
func (b *SheetBuilder) WithColumns(columns ...ColumnInfo) *SheetBuilder {
	b.columns = columns
	return b
}

// WithLayout sets layout options
func (b *SheetBuilder) WithLayout(layout *LayoutTemplate) *SheetBuilder {
	b.layout = layout
	return b
}

// WithProtection sets protection options
func (b *SheetBuilder) WithProtection(protection *ProtectionTemplate) *SheetBuilder {
	b.protection = protection
	return b
}

// Build finalizes the sheet and adds it to the exporter
func (b *SheetBuilder) Build() *DataExporter {
	b.exporter.data[b.sheetName] = b.sheetData
	return b.exporter
}
