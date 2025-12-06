package pgexcel

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

// data_exporter.go - Export Go data structures (slices, structs, maps) to Excel
// This is a self-contained exporter for in-memory data (no database dependency)

// =============================================================================
// Core Types
// =============================================================================
const (
	SectionDirectionHorizontal = "horizontal"
	SectionDirectionVertical   = "vertical"
)

// CellStyle defines styling for cells
type CellStyle struct {
	FontName   string
	FontSize   float64
	FontBold   bool
	FontItalic bool
	FontColor  string

	FillColor   string
	FillPattern int

	Alignment     string // "left", "center", "right"
	VerticalAlign string // "top", "middle", "bottom"

	BorderStyle string
	BorderColor string

	NumberFormat string

	WrapText bool
	Locked   bool
}

// DefaultHeaderStyle returns a default style for headers
func DefaultHeaderStyle() *CellStyle {
	return &CellStyle{
		FontName:      "Arial",
		FontSize:      11,
		FontBold:      true,
		FontColor:     "#FFFFFF",
		FillColor:     "#4472C4",
		FillPattern:   1,
		Alignment:     "center",
		VerticalAlign: "middle",
		Locked:        true,
	}
}

// DefaultDataStyle returns a default style for data cells
func DefaultDataStyle() *CellStyle {
	return &CellStyle{
		FontName:      "Arial",
		FontSize:      10,
		Alignment:     "left",
		VerticalAlign: "middle",
		Locked:        true,
	}
}

// SheetProtection holds the protection configuration for a sheet
type SheetProtection struct {
	Password       string
	ProtectSheet   bool
	LockedCells    map[string]bool
	LockedRanges   []CellRange
	UnlockedRanges []CellRange
	LockedColumns  []ColumnRange
	LockedRows     []RowRange

	// Advanced protection options
	AllowFormatCells      bool
	AllowFormatColumns    bool
	AllowFormatRows       bool
	AllowInsertColumns    bool
	AllowInsertRows       bool
	AllowInsertHyperlinks bool
	AllowDeleteColumns    bool
	AllowDeleteRows       bool
	AllowSort             bool
	AllowFilter           bool
	AllowPivotTables      bool
}

// CellRange represents a range of cells in Excel notation (e.g., "A1:B10")
type CellRange struct {
	StartCol string
	StartRow int
	EndCol   string
	EndRow   int
}

// ColumnRange represents one or more columns
type ColumnRange struct {
	Start string
	End   string
}

// RowRange represents one or more rows
type RowRange struct {
	Start int
	End   int
}

// NewSheetProtection creates a new SheetProtection with sensible defaults
func NewSheetProtection() *SheetProtection {
	return &SheetProtection{
		ProtectSheet:          true,
		LockedCells:           make(map[string]bool),
		AllowFormatCells:      false,
		AllowFormatColumns:    false,
		AllowFormatRows:       false,
		AllowInsertColumns:    false,
		AllowInsertRows:       false,
		AllowInsertHyperlinks: false,
		AllowDeleteColumns:    false,
		AllowDeleteRows:       false,
		AllowSort:             false,
		AllowFilter:           true,
		AllowPivotTables:      false,
	}
}

// =============================================================================
// Template Types (YAML-mappable)
// =============================================================================

// ReportTemplate represents the complete YAML template configuration
type ReportTemplate struct {
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Defaults    *TemplateDefaults `yaml:"defaults,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Sheets      []SheetTemplate   `yaml:"sheets"`
}

// TemplateDefaults holds default configurations applied to all sheets
type TemplateDefaults struct {
	HeaderStyle *StyleTemplate `yaml:"header_style,omitempty"`
	DataStyle   *StyleTemplate `yaml:"data_style,omitempty"`
	DateFormat  string         `yaml:"date_format,omitempty"`
	TimeFormat  string         `yaml:"time_format,omitempty"`
	NumFormat   string         `yaml:"number_format,omitempty"`
}

// SheetTemplate represents a single sheet configuration
type SheetTemplate struct {
	Name       string              `yaml:"name"`
	Query      string              `yaml:"query,omitempty"`
	QueryFile  string              `yaml:"query_file,omitempty"`
	QueryArgs  []string            `yaml:"query_args,omitempty"`
	Columns    []ColumnTemplate    `yaml:"columns,omitempty"`
	Sections   []SectionConfig     `yaml:"sections,omitempty"` // For multi-section sheets
	Protection *ProtectionTemplate `yaml:"protection,omitempty"`
	Style      *SheetStyleTemplate `yaml:"style,omitempty"`
	Layout     *LayoutTemplate     `yaml:"layout,omitempty"`
}

// ColumnTemplate defines column-specific configurations
type ColumnTemplate struct {
	Name        string            `yaml:"name"`
	Header      string            `yaml:"header,omitempty"`
	Width       float64           `yaml:"width,omitempty"`
	Format      string            `yaml:"format,omitempty"`
	Style       *StyleTemplate    `yaml:"style,omitempty"`
	Hidden      bool              `yaml:"hidden,omitempty"`
	Formula     string            `yaml:"formula,omitempty"`
	Conditional []ConditionalRule `yaml:"conditional,omitempty"`
}

// ConditionalRule defines conditional formatting based on cell values
type ConditionalRule struct {
	Condition string         `yaml:"condition"`
	Style     *StyleTemplate `yaml:"style"`
}

// ProtectionTemplate defines sheet protection configuration
type ProtectionTemplate struct {
	Password              string   `yaml:"password,omitempty"`
	LockSheet             bool     `yaml:"lock_sheet"`
	LockedColumns         []string `yaml:"locked_columns,omitempty"`
	UnlockedColumns       []string `yaml:"unlocked_columns,omitempty"`
	LockedRows            []string `yaml:"locked_rows,omitempty"`
	UnlockedRanges        []string `yaml:"unlocked_ranges,omitempty"`
	AllowFilter           bool     `yaml:"allow_filter,omitempty"`
	AllowSort             bool     `yaml:"allow_sort,omitempty"`
	AllowFormatCells      bool     `yaml:"allow_format_cells,omitempty"`
	AllowFormatColumns    bool     `yaml:"allow_format_columns,omitempty"`
	AllowFormatRows       bool     `yaml:"allow_format_rows,omitempty"`
	AllowInsertRows       bool     `yaml:"allow_insert_rows,omitempty"`
	AllowInsertColumns    bool     `yaml:"allow_insert_columns,omitempty"`
	AllowInsertHyperlinks bool     `yaml:"allow_insert_hyperlinks,omitempty"`
	AllowDeleteRows       bool     `yaml:"allow_delete_rows,omitempty"`
	AllowDeleteColumns    bool     `yaml:"allow_delete_columns,omitempty"`
	AllowPivotTables      bool     `yaml:"allow_pivot_tables,omitempty"`
}

// SheetStyleTemplate defines sheet-level style overrides
type SheetStyleTemplate struct {
	HeaderStyle *StyleTemplate `yaml:"header_style,omitempty"`
	DataStyle   *StyleTemplate `yaml:"data_style,omitempty"`
}

// LayoutTemplate controls sheet layout options
type LayoutTemplate struct {
	FreezeRows      int    `yaml:"freeze_rows,omitempty"`
	FreezeCols      int    `yaml:"freeze_cols,omitempty"`
	AutoFilter      bool   `yaml:"auto_filter,omitempty"`
	AutoFitCols     bool   `yaml:"auto_fit_columns,omitempty"`
	MaxColWidth     int    `yaml:"max_column_width,omitempty"`
	ShowGridlines   *bool  `yaml:"show_gridlines,omitempty"`
	PrintArea       string `yaml:"print_area,omitempty"`
	PageOrientation string `yaml:"page_orientation,omitempty"`
}

// StyleTemplate for cell/column/header styling
type StyleTemplate struct {
	Font         *FontTemplate   `yaml:"font,omitempty"`
	Fill         *FillTemplate   `yaml:"fill,omitempty"`
	Border       *BorderTemplate `yaml:"border,omitempty"`
	Alignment    string          `yaml:"alignment,omitempty"`
	VAlignment   string          `yaml:"valignment,omitempty"`
	NumberFormat string          `yaml:"number_format,omitempty"`
	WrapText     bool            `yaml:"wrap_text,omitempty"`
	Locked       *bool           `yaml:"locked,omitempty"`
}

// FontTemplate defines font properties
type FontTemplate struct {
	Name   string  `yaml:"name,omitempty"`
	Size   float64 `yaml:"size,omitempty"`
	Bold   bool    `yaml:"bold,omitempty"`
	Italic bool    `yaml:"italic,omitempty"`
	Color  string  `yaml:"color,omitempty"`
}

// FillTemplate defines cell fill/background
type FillTemplate struct {
	Color   string `yaml:"color,omitempty"`
	Pattern int    `yaml:"pattern,omitempty"`
}

// BorderTemplate defines cell borders
type BorderTemplate struct {
	Style string `yaml:"style,omitempty"`
	Color string `yaml:"color,omitempty"`
}

// GetHeader returns the display header (falls back to column name)
func (c *ColumnTemplate) GetHeader() string {
	if c.Header != "" {
		return c.Header
	}
	return c.Name
}

// IsEmpty checks if a style has any values set
func (s *StyleTemplate) IsEmpty() bool {
	if s == nil {
		return true
	}
	return s.Font == nil && s.Fill == nil && s.Border == nil &&
		s.Alignment == "" && s.VAlignment == "" && s.NumberFormat == "" &&
		!s.WrapText && s.Locked == nil
}

// ToCellStyle converts StyleTemplate to CellStyle
func (s *StyleTemplate) ToCellStyle() *CellStyle {
	if s == nil || s.IsEmpty() {
		return nil
	}

	style := &CellStyle{
		Alignment:     s.Alignment,
		VerticalAlign: s.VAlignment,
		NumberFormat:  s.NumberFormat,
		WrapText:      s.WrapText,
	}

	if s.Font != nil {
		style.FontName = s.Font.Name
		style.FontSize = s.Font.Size
		style.FontBold = s.Font.Bold
		style.FontItalic = s.Font.Italic
		style.FontColor = s.Font.Color
	}

	if s.Fill != nil {
		style.FillColor = s.Fill.Color
		style.FillPattern = s.Fill.Pattern
		if style.FillPattern == 0 && style.FillColor != "" {
			style.FillPattern = 1
		}
	}

	if s.Border != nil {
		style.BorderStyle = s.Border.Style
		style.BorderColor = s.Border.Color
	}

	if s.Locked != nil {
		style.Locked = *s.Locked
	}

	return style
}

// Merge merges another StyleTemplate into this one (other takes precedence)
func (s *StyleTemplate) Merge(other *StyleTemplate) *StyleTemplate {
	if other == nil {
		return s
	}
	if s == nil {
		return other
	}

	result := &StyleTemplate{
		Alignment:    s.Alignment,
		VAlignment:   s.VAlignment,
		NumberFormat: s.NumberFormat,
		WrapText:     s.WrapText,
		Locked:       s.Locked,
	}

	// Merge font
	if s.Font != nil || other.Font != nil {
		result.Font = &FontTemplate{}
		if s.Font != nil {
			*result.Font = *s.Font
		}
		if other.Font != nil {
			if other.Font.Name != "" {
				result.Font.Name = other.Font.Name
			}
			if other.Font.Size != 0 {
				result.Font.Size = other.Font.Size
			}
			if other.Font.Bold {
				result.Font.Bold = true
			}
			if other.Font.Italic {
				result.Font.Italic = true
			}
			if other.Font.Color != "" {
				result.Font.Color = other.Font.Color
			}
		}
	}

	// Merge fill
	if s.Fill != nil || other.Fill != nil {
		result.Fill = &FillTemplate{}
		if s.Fill != nil {
			*result.Fill = *s.Fill
		}
		if other.Fill != nil {
			if other.Fill.Color != "" {
				result.Fill.Color = other.Fill.Color
			}
			if other.Fill.Pattern != 0 {
				result.Fill.Pattern = other.Fill.Pattern
			}
		}
	}

	// Merge border
	if s.Border != nil || other.Border != nil {
		result.Border = &BorderTemplate{}
		if s.Border != nil {
			*result.Border = *s.Border
		}
		if other.Border != nil {
			if other.Border.Style != "" {
				result.Border.Style = other.Border.Style
			}
			if other.Border.Color != "" {
				result.Border.Color = other.Border.Color
			}
		}
	}

	// Override scalar values from other
	if other.Alignment != "" {
		result.Alignment = other.Alignment
	}
	if other.VAlignment != "" {
		result.VAlignment = other.VAlignment
	}
	if other.NumberFormat != "" {
		result.NumberFormat = other.NumberFormat
	}
	if other.WrapText {
		result.WrapText = true
	}
	if other.Locked != nil {
		result.Locked = other.Locked
	}

	return result
}

// =============================================================================
// Template Loading
// =============================================================================

// LoadTemplate loads a report template from a YAML file
func LoadTemplate(path string) (*ReportTemplate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening template file: %w", err)
	}
	defer file.Close()

	return LoadTemplateFromReader(file)
}

// LoadTemplateFromReader loads a template from an io.Reader
func LoadTemplateFromReader(r io.Reader) (*ReportTemplate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	var template ReportTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("parsing YAML template: %w", err)
	}

	if err := template.applyDefaults(); err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	if err := ValidateTemplate(&template); err != nil {
		return nil, fmt.Errorf("validating template: %w", err)
	}

	return &template, nil
}

// LoadTemplateFromString loads a template from a YAML string
func LoadTemplateFromString(yamlContent string) (*ReportTemplate, error) {
	return LoadTemplateFromReader(strings.NewReader(yamlContent))
}

// ValidateTemplate validates the template structure
func ValidateTemplate(t *ReportTemplate) error {
	if t == nil {
		return fmt.Errorf("template is nil")
	}

	if len(t.Sheets) == 0 {
		return fmt.Errorf("template must have at least one sheet")
	}

	for i, sheet := range t.Sheets {
		if err := validateSheet(&sheet, i); err != nil {
			return err
		}
	}

	return nil
}

func validateSheet(s *SheetTemplate, index int) error {
	if s.Name == "" {
		return fmt.Errorf("sheet[%d]: name is required", index)
	}

	// Either query/query_file OR sections must be specified (but not both)
	hasQuery := s.Query != "" || s.QueryFile != ""
	hasSections := len(s.Sections) > 0

	if !hasQuery && !hasSections {
		return fmt.Errorf("sheet[%d] '%s': either query, query_file, or sections is required", index, s.Name)
	}

	if hasQuery && hasSections {
		return fmt.Errorf("sheet[%d] '%s': cannot specify both query/query_file and sections", index, s.Name)
	}

	if s.Query != "" && s.QueryFile != "" {
		return fmt.Errorf("sheet[%d] '%s': cannot specify both query and query_file", index, s.Name)
	}

	colNames := make(map[string]bool)
	for j, col := range s.Columns {
		if col.Name == "" {
			return fmt.Errorf("sheet[%d] '%s' column[%d]: name is required", index, s.Name, j)
		}
		if colNames[col.Name] {
			return fmt.Errorf("sheet[%d] '%s': duplicate column name '%s'", index, s.Name, col.Name)
		}
		colNames[col.Name] = true
	}

	if s.Protection != nil {
		if err := validateProtection(s.Protection, s.Name); err != nil {
			return err
		}
	}

	return nil
}

func validateProtection(p *ProtectionTemplate, sheetName string) error {
	for _, rng := range p.UnlockedRanges {
		if !isValidCellRange(rng) {
			return fmt.Errorf("sheet '%s': invalid range format '%s' (expected A1:B10)", sheetName, rng)
		}
	}

	for _, row := range p.LockedRows {
		if row != "header" && !isValidRowRange(row) {
			return fmt.Errorf("sheet '%s': invalid row range '%s' (expected number, range like 1-5, or 'header')", sheetName, row)
		}
	}

	return nil
}

func isValidCellRange(s string) bool {
	pattern := `^[A-Z]+\d+:[A-Z]+\d+$`
	matched, _ := regexp.MatchString(pattern, strings.ToUpper(s))
	return matched
}

func isValidRowRange(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	parts := strings.Split(s, "-")
	if len(parts) == 2 {
		_, err1 := strconv.Atoi(parts[0])
		_, err2 := strconv.Atoi(parts[1])
		return err1 == nil && err2 == nil
	}

	return false
}

func (t *ReportTemplate) applyDefaults() error {
	if t.Version == "" {
		t.Version = "1.0"
	}

	if t.Variables == nil {
		t.Variables = make(map[string]string)
	}

	for i := range t.Sheets {
		if err := t.Sheets[i].applyDefaults(t.Defaults); err != nil {
			return fmt.Errorf("sheet '%s': %w", t.Sheets[i].Name, err)
		}
	}

	return nil
}

func (s *SheetTemplate) applyDefaults(defaults *TemplateDefaults) error {
	if s.Layout == nil {
		s.Layout = &LayoutTemplate{}
	}

	if s.Style == nil && defaults != nil {
		s.Style = &SheetStyleTemplate{
			HeaderStyle: defaults.HeaderStyle,
			DataStyle:   defaults.DataStyle,
		}
	} else if s.Style != nil && defaults != nil {
		if s.Style.HeaderStyle == nil {
			s.Style.HeaderStyle = defaults.HeaderStyle
		}
		if s.Style.DataStyle == nil {
			s.Style.DataStyle = defaults.DataStyle
		}
	}

	for i := range s.Columns {
		if s.Columns[i].Header == "" {
			s.Columns[i].Header = s.Columns[i].Name
		}
	}

	return nil
}

// GetColumnByName finds a column template by database column name
func (s *SheetTemplate) GetColumnByName(name string) *ColumnTemplate {
	for i := range s.Columns {
		if s.Columns[i].Name == name {
			return &s.Columns[i]
		}
	}
	return nil
}

// ToSheetProtection converts ProtectionTemplate to SheetProtection
func (p *ProtectionTemplate) ToSheetProtection() *SheetProtection {
	if p == nil || !p.LockSheet {
		return nil
	}

	sp := NewSheetProtection()
	sp.Password = p.Password
	sp.ProtectSheet = p.LockSheet
	sp.AllowFilter = p.AllowFilter
	sp.AllowSort = p.AllowSort
	sp.AllowFormatCells = p.AllowFormatCells
	sp.AllowFormatColumns = p.AllowFormatColumns
	sp.AllowFormatRows = p.AllowFormatRows
	sp.AllowInsertRows = p.AllowInsertRows
	sp.AllowInsertColumns = p.AllowInsertColumns
	sp.AllowInsertHyperlinks = p.AllowInsertHyperlinks
	sp.AllowDeleteRows = p.AllowDeleteRows
	sp.AllowDeleteColumns = p.AllowDeleteColumns
	sp.AllowPivotTables = p.AllowPivotTables

	return sp
}

// =============================================================================
// DataExporter - Main Export Logic
// =============================================================================

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

// NewDataExporterFromYaml creates a data exporter from a YAML template string
// This allows the caller to manage reading the template from any source (file, database, embedded, etc.)
func NewDataExporterFromYaml(yamlContent string) (*DataExporter, error) {
	template, err := LoadTemplateFromString(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}
	return NewDataExporterWithTemplate(template), nil
}

// NewDataExporterFromYamlFile creates a data exporter from a YAML template file
func NewDataExporterFromYamlFile(filepath string) (*DataExporter, error) {
	template, err := LoadTemplate(filepath)
	if err != nil {
		return nil, fmt.Errorf("loading template file: %w", err)
	}
	return NewDataExporterWithTemplate(template), nil
}

// WithData adds data for a sheet (data should be a slice of structs or maps)
func (e *DataExporter) WithData(sheetName string, data interface{}) *DataExporter {
	e.data[sheetName] = data
	return e
}

// BindSectionData binds data to a section identified by its ID in the YAML template
// The sectionID must match a section with the corresponding "id" field in the template
func (e *DataExporter) BindSectionData(sectionID string, data interface{}) *DataExporter {
	if e.template == nil {
		return e
	}

	// Find the section across all sheets and bind the data
	for sheetIdx := range e.template.Sheets {
		for sectionIdx := range e.template.Sheets[sheetIdx].Sections {
			if e.template.Sheets[sheetIdx].Sections[sectionIdx].ID == sectionID {
				e.template.Sheets[sheetIdx].Sections[sectionIdx].Data = data
				return e
			}
		}
	}

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
	processedSheets := make(map[string]bool)

	// First, process template-defined sheets with sections
	if e.template != nil {
		for _, sheetTmpl := range e.template.Sheets {
			if len(sheetTmpl.Sections) > 0 {
				// Convert template sections to sheetWithSections
				sections := make([]*SectionConfig, len(sheetTmpl.Sections))
				for i := range sheetTmpl.Sections {
					sections[i] = &sheetTmpl.Sections[i]
				}

				sws := &sheetWithSections{
					sections:   sections,
					layout:     sheetTmpl.Layout,
					protection: sheetTmpl.Protection,
				}

				if err := e.exportSections(f, sheetTmpl.Name, sws, sheetIdx == 0); err != nil {
					return fmt.Errorf("exporting sections sheet '%s': %w", sheetTmpl.Name, err)
				}
				processedSheets[sheetTmpl.Name] = true
				sheetIdx++
			}
		}
	}

	// Then, process data added via WithData or AddSheet builder
	for sheetName, data := range e.data {
		if processedSheets[sheetName] {
			continue // Already processed via template sections
		}

		// Check if this is a section-based sheet (from builder pattern)
		if sws, ok := data.(*sheetWithSections); ok {
			if err := e.exportSections(f, sheetName, sws, sheetIdx == 0); err != nil {
				return fmt.Errorf("exporting sections sheet '%s': %w", sheetName, err)
			}
			processedSheets[sheetName] = true
			sheetIdx++
			continue
		}

		// Regular single-data sheet export
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
		processedSheets[sheetName] = true
		sheetIdx++
	}

	// Delete default Sheet1 if we created other sheets and didn't use it
	if len(processedSheets) > 0 {
		if !processedSheets["Sheet1"] {
			f.DeleteSheet("Sheet1")
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
			if len(col.Conditional) > 0 {
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
// For time.Time values, we keep them as-is - the Format field is used for Excel cell number format
func (e *DataExporter) formatDataValue(value interface{}, col ColumnInfo) interface{} {
	if value == nil {
		return ""
	}

	// Handle time.Time - keep as time.Time, don't convert to string
	// The format will be applied via cell style (number format)
	if t, ok := value.(time.Time); ok {
		if t.IsZero() {
			return ""
		}
		return t // Return time.Time directly, Excel number format handles display
	}

	// Handle *time.Time
	if t, ok := value.(*time.Time); ok {
		if t == nil || t.IsZero() {
			return ""
		}
		return *t
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

// isTimeValue checks if a value is a time.Time
func isTimeValue(value interface{}) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case time.Time, *time.Time:
		return true
	}
	return false
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

// toFloat64 converts numeric types to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	}
	return 0
}

// =============================================================================
// Section-based Export Types
// =============================================================================

// SectionConfig defines a data section within a sheet
// Each section can have different columns, styling, and protection settings
// Supports both programmatic configuration and YAML loading
type SectionConfig struct {
	ID         string      `yaml:"id,omitempty"`          // Identifier for binding data from YAML
	Title      string      `yaml:"title,omitempty"`       // Optional section title row
	Data       interface{} `yaml:"-"`                     // Slice of structs or maps (runtime only)
	ShowHeader bool        `yaml:"show_header,omitempty"` // Show column headers (default: true if not set)
	Locked     bool        `yaml:"locked,omitempty"`      // Lock this section from editing
	GapAfter   int         `yaml:"gap_after,omitempty"`   // Gap after section: blank rows (vertical) or blank columns (horizontal)

	// Layout direction - specifies how this section is positioned relative to the previous section
	// "vertical" (default): section is placed BELOW the previous section
	// "horizontal": section is placed TO THE RIGHT of the previous section
	Direction   string `yaml:"direction,omitempty"`    // "vertical" (default) or "horizontal"
	Position    string `yaml:"position,omitempty"`     // Excel-style position (e.g., "A1", "B3"). Overrides StartColumn/StartRow if both are set
	StartColumn int    `yaml:"start_column,omitempty"` // Starting column index (0-based), auto-calculated if not set
	StartRow    int    `yaml:"start_row,omitempty"`    // Starting row (1-based), uses current row if not set

	// Styling
	TitleStyle  *StyleTemplate `yaml:"title_style,omitempty"`  // Style for title row
	HeaderStyle *StyleTemplate `yaml:"header_style,omitempty"` // Style for column headers
	DataStyle   *StyleTemplate `yaml:"data_style,omitempty"`   // Style for data cells

	// Column customization (optional - defaults from struct tags)
	Columns []ColumnConfig `yaml:"columns,omitempty"` // Override column headers/widths/formats
}

// ColumnConfig allows per-section column customization
// Supports both programmatic configuration and YAML loading
type ColumnConfig struct {
	FieldName string  `yaml:"field_name"`       // Struct field name or map key
	Header    string  `yaml:"header,omitempty"` // Display header
	Width     float64 `yaml:"width,omitempty"`  // Column width
	Format    string  `yaml:"format,omitempty"` // Number/date format
	Hidden    bool    `yaml:"hidden,omitempty"` // Hide this column
}

// SheetBuilder provides a fluent API for building sheets
type SheetBuilder struct {
	exporter   *DataExporter
	sheetName  string
	sheetData  interface{}
	columns    []ColumnInfo
	layout     *LayoutTemplate
	protection *ProtectionTemplate
	sections   []*SectionConfig // Sections for stacked data
}

// WithData sets the data for this sheet (single data mode)
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

// WithProtection sets protection options (sheet-level)
func (b *SheetBuilder) WithProtection(protection *ProtectionTemplate) *SheetBuilder {
	b.protection = protection
	return b
}

// AddSection adds a data section to this sheet (supports stacking multiple collections)
func (b *SheetBuilder) AddSection(config *SectionConfig) *SheetBuilder {
	if config != nil {
		b.sections = append(b.sections, config)
	}
	return b
}

// Build finalizes the sheet and adds it to the exporter
func (b *SheetBuilder) Build() *DataExporter {
	if len(b.sections) > 0 {
		// Store sections for section-based export
		b.exporter.data[b.sheetName] = &sheetWithSections{
			sections:   b.sections,
			layout:     b.layout,
			protection: b.protection,
		}
	} else if b.sheetData != nil {
		// Store regular data for single-data export
		b.exporter.data[b.sheetName] = b.sheetData
	}
	return b.exporter
}

// sheetWithSections is an internal type to mark section-based sheets
type sheetWithSections struct {
	sections   []*SectionConfig
	layout     *LayoutTemplate
	protection *ProtectionTemplate
}

// exportSections exports multiple sections with support for vertical and horizontal stacking
func (e *DataExporter) exportSections(f *excelize.File, sheetName string, sws *sheetWithSections, isFirst bool) error {
	// Create or rename sheet
	if isFirst {
		f.SetSheetName("Sheet1", sheetName)
	} else {
		if _, err := f.NewSheet(sheetName); err != nil {
			return fmt.Errorf("creating sheet: %w", err)
		}
	}

	// Track positions for layout
	currentRow := 1        // For vertical sections - next row to place content
	maxRow := 1            // Track maximum row used
	maxCol := 0            // Track maximum column used by any section
	prevSectionEndCol := 0 // Track where the previous section ended (for horizontal placement)
	hasLockedSections := false
	hasUnlockedSections := false

	// First pass: identify horizontal section groups (consecutive horizontal sections)
	// and process all sections
	for _, section := range sws.sections {
		if section.Data == nil {
			continue
		}

		// Get data as slice using reflection
		dataVal := reflect.ValueOf(section.Data)
		if dataVal.Kind() == reflect.Ptr {
			dataVal = dataVal.Elem()
		}
		if dataVal.Kind() != reflect.Slice {
			return fmt.Errorf("section data must be a slice, got %s", dataVal.Kind())
		}

		if dataVal.Len() == 0 && section.Title == "" {
			continue
		}

		// Extract columns for this section
		firstRow := dataVal.Index(0)
		columns, colErr := e.extractColumnsForSection(firstRow, section)
		if colErr != nil {
			return fmt.Errorf("extracting columns for section: %w", colErr)
		}

		// Track locked/unlocked
		if section.Locked {
			hasLockedSections = true
		} else {
			hasUnlockedSections = true
		}

		// Create styles
		titleStyleID, headerStyleID, _, styleErr := e.createSectionStyles(f, section)
		if styleErr != nil {
			return fmt.Errorf("creating section styles: %w", styleErr)
		}

		// Determine layout direction (default to vertical if not specified)
		direction := section.Direction
		if direction == "" {
			direction = SectionDirectionVertical
		}
		isHorizontal := direction == SectionDirectionHorizontal

		// Calculate starting position based on direction and explicit positioning
		// Priority: Position (Excel-style) > StartColumn/StartRow > Automatic positioning
		var startCol, startRow int

		// First check for Excel-style position (e.g., "B3")
		if section.Position != "" {
			var posErr error
			startCol, startRow, posErr = parseExcelPosition(section.Position)
			if posErr != nil {
				return fmt.Errorf("invalid position '%s': %w", section.Position, posErr)
			}
		} else {
			// Fall back to separate StartRow/StartColumn if no Excel-style position is provided
			if section.StartRow > 0 {
				startRow = section.StartRow
			} else if isHorizontal {
				// Horizontal sections default to row 1 if no explicit row is set
				startRow = 1
			} else {
				// Vertical sections stack below previous content
				startRow = maxRow
			}

			if section.StartColumn > 0 {
				// Use explicit column if provided
				startCol = section.StartColumn
			} else if isHorizontal {
				// Horizontal sections stack to the right of previous content
				startCol = prevSectionEndCol
			} else {
				// Vertical sections start at column 0
				startCol = 0
			}
		}

		sectionRow := startRow

		// Write section title if provided
		if section.Title != "" {
			cell := columnIndexToName(startCol) + fmt.Sprintf("%d", sectionRow)
			if err := f.SetCellValue(sheetName, cell, section.Title); err != nil {
				return fmt.Errorf("setting title: %w", err)
			}
			if titleStyleID != 0 {
				// Merge title across all columns
				endCell := columnIndexToName(startCol+len(columns)-1) + fmt.Sprintf("%d", sectionRow)
				f.MergeCell(sheetName, cell, endCell)
				if err := f.SetCellStyle(sheetName, cell, endCell, titleStyleID); err != nil {
					return fmt.Errorf("setting title style: %w", err)
				}
			} else if section.Locked {
				e.applyCellLock(f, sheetName, cell, true)
			}
			sectionRow++
		}

		// Write headers
		showHeader := section.ShowHeader || (section.Title == "" && !section.ShowHeader)
		if len(columns) > 0 && showHeader {
			for colIdx, col := range columns {
				cell := columnIndexToName(startCol+colIdx) + fmt.Sprintf("%d", sectionRow)
				if err := f.SetCellValue(sheetName, cell, col.Header); err != nil {
					return fmt.Errorf("setting header: %w", err)
				}
				if headerStyleID != 0 {
					if err := f.SetCellStyle(sheetName, cell, cell, headerStyleID); err != nil {
						return fmt.Errorf("setting header style: %w", err)
					}
				} else if section.Locked {
					e.applyCellLock(f, sheetName, cell, true)
				}
			}
			sectionRow++
		}

		// Write data rows
		for rowIdx := 0; rowIdx < dataVal.Len(); rowIdx++ {
			rowVal := dataVal.Index(rowIdx)

			for colIdx, col := range columns {
				cell := columnIndexToName(startCol+colIdx) + fmt.Sprintf("%d", sectionRow)
				value := e.getFieldValue(rowVal, col.FieldName)
				displayValue := e.formatDataValue(value, col)

				if err := f.SetCellValue(sheetName, cell, displayValue); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}

				// Create a combined style with format (if any) and protection
				cellStyle := &excelize.Style{
					Protection: &excelize.Protection{
						Locked: section.Locked,
					},
				}

				// Add number format if column has one
				if col.Format != "" {
					cellStyle.CustomNumFmt = &col.Format
				}

				styleID, err := f.NewStyle(cellStyle)
				if err != nil {
					return fmt.Errorf("creating cell style: %w", err)
				}

				if err := f.SetCellStyle(sheetName, cell, cell, styleID); err != nil {
					return fmt.Errorf("setting cell style: %w", err)
				}
			}
			sectionRow++
		}

		// Set column widths
		for colIdx, col := range columns {
			if col.Width > 0 {
				colName := columnIndexToName(startCol + colIdx)
				f.SetColWidth(sheetName, colName, colName, col.Width)
			}
		}

		// Track where this section ended (column-wise)
		sectionEndCol := startCol + len(columns) + section.GapAfter

		// Update position trackers based on direction
		if isHorizontal {
			// Move column position to the right for next horizontal section
			prevSectionEndCol = sectionEndCol
			if sectionRow > maxRow {
				maxRow = sectionRow
			}
		} else {
			// Move row position down for next vertical section
			currentRow = sectionRow + section.GapAfter
			if currentRow > maxRow {
				maxRow = currentRow
			}
			// Update prevSectionEndCol so horizontal sections know where to start
			prevSectionEndCol = sectionEndCol
		}

		// Track max column
		if sectionEndCol > maxCol {
			maxCol = sectionEndCol
		}
	}

	// Apply layout if provided
	if sws.layout != nil {
		if err := e.applyLayout(f, sheetName, maxCol, maxRow, sws.layout); err != nil {
			return fmt.Errorf("applying layout: %w", err)
		}
	}

	// Apply sheet protection if there are locked sections
	if hasLockedSections {
		protectOpts := &excelize.SheetProtectionOptions{
			SelectLockedCells:   true,
			SelectUnlockedCells: true,
		}
		if sws.protection != nil && sws.protection.Password != "" {
			protectOpts.Password = sws.protection.Password
		}
		if sws.protection != nil {
			protectOpts.AutoFilter = sws.protection.AllowFilter
			protectOpts.Sort = sws.protection.AllowSort
		}
		if err := f.ProtectSheet(sheetName, protectOpts); err != nil {
			return fmt.Errorf("protecting sheet: %w", err)
		}
	}

	_ = hasUnlockedSections
	return nil
}

// extractColumnsForSection extracts column info with section-level overrides
func (e *DataExporter) extractColumnsForSection(val reflect.Value, section *SectionConfig) ([]ColumnInfo, error) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var columns []ColumnInfo

	switch val.Kind() {
	case reflect.Struct:
		columns = e.extractColumnsFromStruct(val, nil)
	case reflect.Map:
		columns = e.extractColumnsFromMap(val, nil)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", val.Kind())
	}

	// Apply section-level column overrides
	if len(section.Columns) > 0 {
		overrideMap := make(map[string]*ColumnConfig)
		for i := range section.Columns {
			overrideMap[section.Columns[i].FieldName] = &section.Columns[i]
		}

		for i := range columns {
			if override, ok := overrideMap[columns[i].FieldName]; ok {
				if override.Header != "" {
					columns[i].Header = override.Header
				}
				if override.Width > 0 {
					columns[i].Width = override.Width
				}
				if override.Format != "" {
					columns[i].Format = override.Format
				}
				columns[i].Hidden = override.Hidden
			}
		}
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

// createSectionStyles creates styles for a section
func (e *DataExporter) createSectionStyles(f *excelize.File, section *SectionConfig) (titleID, headerID, dataID int, err error) {
	// Helper to enforce locked status
	enforceLocked := func(tmpl *StyleTemplate) *StyleTemplate {
		if !section.Locked {
			return tmpl
		}
		if tmpl == nil {
			tmpl = &StyleTemplate{}
		}
		locked := true
		tmpl.Locked = &locked
		return tmpl
	}

	enforceLockedStyle := func(style *CellStyle) *CellStyle {
		if !section.Locked {
			return style
		}
		if style == nil {
			return &CellStyle{Locked: true}
		}
		style.Locked = true
		return style
	}

	// Title style
	if section.TitleStyle != nil {
		titleID, err = e.createStyleFromTemplate(f, enforceLocked(section.TitleStyle))
		if err != nil {
			return 0, 0, 0, err
		}
	} else if section.Title != "" {
		// Default title style
		titleID, err = e.createStyleFromCellStyle(f, enforceLockedStyle(&CellStyle{
			FontName:    "Arial",
			FontSize:    12,
			FontBold:    true,
			FillColor:   "#E0E0E0",
			FillPattern: 1,
			Alignment:   "left",
		}))
		if err != nil {
			return 0, 0, 0, err
		}
	}

	// Header style
	if section.HeaderStyle != nil {
		headerID, err = e.createStyleFromTemplate(f, enforceLocked(section.HeaderStyle))
		if err != nil {
			return 0, 0, 0, err
		}
	} else {
		headerID, err = e.createStyleFromCellStyle(f, enforceLockedStyle(DefaultHeaderStyle()))
		if err != nil {
			return 0, 0, 0, err
		}
	}

	// Data style
	if section.DataStyle != nil {
		dataID, err = e.createStyleFromTemplate(f, enforceLocked(section.DataStyle))
		if err != nil {
			return 0, 0, 0, err
		}
	} else {
		dataID, err = e.createStyleFromCellStyle(f, enforceLockedStyle(DefaultDataStyle()))
		if err != nil {
			return 0, 0, 0, err
		}
	}

	return titleID, headerID, dataID, nil
}

// applyCellLock sets the locked property of a cell
func (e *DataExporter) applyCellLock(f *excelize.File, sheetName, cell string, locked bool) {
	style, _ := f.NewStyle(&excelize.Style{
		Protection: &excelize.Protection{
			Locked: locked,
		},
	})
	f.SetCellStyle(sheetName, cell, cell, style)
}

// createColumnFormatStyles creates styles for columns that have custom formats (e.g., dates)
// Returns a map of column index -> style ID
func (e *DataExporter) createColumnFormatStyles(f *excelize.File, columns []ColumnInfo) (map[int]int, error) {
	colStyles := make(map[int]int)

	for i, col := range columns {
		if col.Format != "" {
			styleID, err := f.NewStyle(&excelize.Style{
				CustomNumFmt: &col.Format,
			})
			if err != nil {
				return nil, fmt.Errorf("creating format style for column %s: %w", col.FieldName, err)
			}
			colStyles[i] = styleID
		}
	}

	return colStyles, nil
}

// =============================================================================
// Utilities
// =============================================================================

// columnIndexToName converts column index (0-based) to Excel column name
func columnIndexToName(index int) string {
	if index < 0 {
		return ""
	}
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var result string
	index++ // Convert to 1-based
	for index > 0 {
		index--
		result = string(letters[index%26]) + result
		index = index / 26
	}
	return result
}

// parseExcelPosition parses an Excel-style cell reference (e.g., "A1") into column and row numbers.
// Returns column (0-based) and row (1-based) numbers.
func parseExcelPosition(pos string) (col int, row int, err error) {
	re := regexp.MustCompile(`^([A-Za-z]+)([1-9]\d*)$`)
	matches := re.FindStringSubmatch(pos)
	if matches == nil {
		return 0, 0, fmt.Errorf("invalid Excel position format: %s, expected format like 'A1' or 'B3'", pos)
	}

	// Parse column (A=0, B=1, ..., Z=25, AA=26, etc.)
	colStr := strings.ToUpper(matches[1])
	col = 0
	for i := 0; i < len(colStr); i++ {
		col = col*26 + int(colStr[i]-'A'+1)
	}
	col-- // Convert to 0-based

	// Parse row (1-based)
	rowNum, convErr := strconv.Atoi(matches[2])
	if convErr != nil || rowNum < 1 {
		return 0, 0, fmt.Errorf("invalid row number in position: %s", pos)
	}

	return col, rowNum, nil
}
