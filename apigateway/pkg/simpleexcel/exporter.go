package simpleexcel

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Constants & Types
// =============================================================================

const (
	SectionDirectionHorizontal = "horizontal"
	SectionDirectionVertical   = "vertical"
	SectionTypeFull            = "full"   // Normal section with title, header, and data
	SectionTypeTitleOnly       = "title"  // Only display title
	SectionTypeHidden          = "hidden" // Hidden section (row will be hidden)
	DefaultLockedColor         = "E0E0E0" // Light Gray for locked cells
)

// DataExporter is the main entry point for exporting data.
type DataExporter struct {
	template *ReportTemplate
	// data holds data bound to specific section IDs (for YAML flow)
	data map[string]interface{}
	// sheets holds manually added sheets (for programmatic flow)
	sheets []*SheetBuilder
	// formatters holds registered formatter functions by name
	formatters map[string]func(interface{}) interface{}
}

// ReportTemplate represents the YAML structure.
type ReportTemplate struct {
	Sheets []SheetTemplate `yaml:"sheets"`
}

// SheetTemplate represents a sheet in the YAML.
type SheetTemplate struct {
	Name     string          `yaml:"name"`
	Sections []SectionConfig `yaml:"sections"`
}

// SectionConfig defines a section of data in a sheet.
type SectionConfig struct {
	ID          string         `yaml:"id"`
	Title       string         `yaml:"title"`
	ColSpan     int            `yaml:"col_span"` // Number of columns to span for title-only sections
	Data        interface{}    `yaml:"-"`        // Data is bound at runtime
	Type        string         `yaml:"type"`     // "full", "title", "hidden"
	Locked      bool           `yaml:"locked"`   // Section-level lock (default for all columns)
	ShowHeader  bool           `yaml:"show_header"`
	Direction   string         `yaml:"direction"` // "horizontal" or "vertical"
	Position    string         `yaml:"position"`  // e.g., "A1"
	TitleStyle  *StyleTemplate `yaml:"title_style"`
	HeaderStyle *StyleTemplate `yaml:"header_style"`
	DataStyle   *StyleTemplate `yaml:"data_style"`
	Columns     []ColumnConfig `yaml:"columns"`
}

// ColumnConfig defines a column in a section.
type ColumnConfig struct {
	FieldName       string                        `yaml:"field_name"` // Struct field name or map key
	Header          string                        `yaml:"header"`
	Width           float64                       `yaml:"width"`
	Locked          *bool                         `yaml:"locked"`            // Column-level lock override (overrides section Locked)
	Formatter       func(interface{}) interface{} `yaml:"-"`                 // Optional custom formatter function (Programmatic)
	FormatterName   string                        `yaml:"formatter"`         // Name of registered formatter (YAML)
	HiddenFieldName string                        `yaml:"hidden_field_name"` // Hidden field name for backend use
}

// IsLocked returns whether this column should be locked.
// If column has explicit Locked setting, use that; otherwise use section default.
func (c *ColumnConfig) IsLocked(sectionLocked bool) bool {
	if c.Locked != nil {
		return *c.Locked
	}
	return sectionLocked
}

// StyleTemplate defines basic styling.
type StyleTemplate struct {
	Font   *FontTemplate `yaml:"font"`
	Fill   *FillTemplate `yaml:"fill"`
	Locked *bool         `yaml:"locked"`
}

type FontTemplate struct {
	Bold  bool   `yaml:"bold"`
	Color string `yaml:"color"` // Hex color
}

type FillTemplate struct {
	Color string `yaml:"color"` // Hex color
}

// =============================================================================
// Constructors
// =============================================================================

func NewDataExporter() *DataExporter {
	return &DataExporter{
		data:       make(map[string]interface{}),
		sheets:     []*SheetBuilder{},
		formatters: make(map[string]func(interface{}) interface{}),
	}
}

func NewDataExporterFromYamlConfig(yamlConfig string) (*DataExporter, error) {
	var tmpl ReportTemplate
	if yamlConfig == "" {
		return nil, fmt.Errorf("yaml config is empty")
	}
	if err := yaml.Unmarshal([]byte(yamlConfig), &tmpl); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	exporter := &DataExporter{
		template:   &tmpl,
		data:       make(map[string]interface{}),
		formatters: make(map[string]func(interface{}) interface{}),
		sheets:     make([]*SheetBuilder, 0),
	}

	// Initialize sheets from template
	for i := range tmpl.Sheets {
		sheetTmpl := &tmpl.Sheets[i]
		sb := &SheetBuilder{
			exporter: exporter,
			name:     sheetTmpl.Name,
			sections: make([]*SectionConfig, len(sheetTmpl.Sections)),
		}
		for j := range sheetTmpl.Sections {
			sb.sections[j] = &sheetTmpl.Sections[j]
		}
		exporter.sheets = append(exporter.sheets, sb)
	}

	return exporter, nil
}

// =============================================================================
// Fluent API
// =============================================================================

// AddSheet starts a new sheet builder.
func (e *DataExporter) AddSheet(name string) *SheetBuilder {
	sb := &SheetBuilder{
		exporter: e,
		name:     name,
		sections: []*SectionConfig{},
	}
	e.sheets = append(e.sheets, sb)
	return sb
}

// BindSectionData binds data to a section ID (for YAML-based export).
func (e *DataExporter) BindSectionData(id string, data interface{}) *DataExporter {
	e.data[id] = data
	return e
}

// RegisterFormatter registers a formatter function with a name.
// This allows referencing formatters by name in YAML configurations.
func (e *DataExporter) RegisterFormatter(name string, f func(interface{}) interface{}) *DataExporter {
	e.formatters[name] = f
	return e
}

// GetSheet returns a SheetBuilder by name, or nil if not found.
func (e *DataExporter) GetSheet(name string) *SheetBuilder {
	for _, sheet := range e.sheets {
		if sheet.name == name {
			return sheet
		}
	}
	return nil
}

// GetSheetByIndex returns a SheetBuilder by index (0-based), or nil if out of bounds.
func (e *DataExporter) GetSheetByIndex(index int) *SheetBuilder {
	if index < 0 || index >= len(e.sheets) {
		return nil
	}
	return e.sheets[index]
}

// BuildExcel constructs an Excel file (*excelize.File) based on the exporter's configuration and data.
// It processes both programmatically added sheets and sheets defined in a YAML template,
// returning the generated excelize.File instance or an error// BuildExcel generates the excel file
func (e *DataExporter) BuildExcel() (*excelize.File, error) {
	f := excelize.NewFile()

	// Process All Sheets (both fluent and YAML-initialized are now in e.sheets)
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			// Check if sheet exists to avoid error if duplicates (though logic shouldn't produce duplicates easily)
			idx, _ := f.GetSheetIndex(sheetName)
			if idx == -1 {
				f.NewSheet(sheetName)
			}
		}

		// Perform Late Binding for any section that has an ID and matching data in e.data
		for _, sec := range sb.sections {
			if sec.ID != "" {
				if data, ok := e.data[sec.ID]; ok {
					sec.Data = data
				}
			}
		}

		if err := e.renderSections(f, sheetName, sb.sections); err != nil {
			return nil, err
		}
	}

	return f, nil
}

// ExportToExcel generates the Excel file on disk.
func (e *DataExporter) ExportToExcel(ctx context.Context, path string) error {
	f, err := e.BuildExcel()
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SaveAs(path)
}

// ToBytes exports the Excel file to an in-memory byte slice.
func (e *DataExporter) ToBytes() ([]byte, error) {
	f, err := e.BuildExcel()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Create a buffer and write the Excel file to it
	buf := new(bytes.Buffer)
	if _, err := f.WriteTo(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// =============================================================================
// SheetBuilder
// =============================================================================

type SheetBuilder struct {
	exporter *DataExporter
	name     string
	sections []*SectionConfig
}

func (sb *SheetBuilder) AddSection(config *SectionConfig) *SheetBuilder {
	sb.sections = append(sb.sections, config)
	return sb
}

func (sb *SheetBuilder) Build() *DataExporter {
	return sb.exporter
}

// =============================================================================
// Rendering Logic
// =============================================================================

func (e *DataExporter) renderSections(f *excelize.File, sheet string, sections []*SectionConfig) error {
	// Trackers for layout
	maxRow := 1            // Next available row for Vertical sections (1-based)
	nextColHorizontal := 1 // Next available col for Horizontal sections (1-based)

	hasLockedCells := false
	hiddenRows := []int{} // Track rows to hide

	for _, sec := range sections {
		// Check if any cell needs locking
		if sec.Locked {
			hasLockedCells = true
		} else {
			// Determine effective columns merging user config and data fields
			finalColumns := mergeColumns(sec.Data, sec.Columns)
			sec.Columns = finalColumns // Update section columns to use the merged list

			// Check if any cell needs locking
			if sec.Locked {
				hasLockedCells = true
			} else {
				// Check if any column is explicitly locked
				for _, col := range sec.Columns {
					if col.Locked != nil && *col.Locked {
						hasLockedCells = true
						break
					}
				}
			}
		}
	}

	// If locking is needed, first UNLOCK all cells by default so user can edit unused cells
	if hasLockedCells {
		unlocked := false
		defaultStyle := &StyleTemplate{
			Locked: &unlocked,
		}
		styleID, _ := createStyle(f, defaultStyle)
		// Apply to all columns roughly (A to XFD is max, but let's do A:XFD)
		// SetColStyle requires col name range.
		f.SetColStyle(sheet, "A:XFD", styleID)
	}

	for _, sec := range sections {
		// Re-determine section type as we iterate again (or we could have stored it)
		// We already processed columns merging in the first loop so sec.Columns is updated.

		// Determine section type
		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeFull
		}

		// Determine Layout
		isHorizontal := sec.Direction == SectionDirectionHorizontal

		startCol := 1
		startRow := 1

		if sec.Position != "" {
			c, r, err := excelize.CellNameToCoordinates(sec.Position)
			if err == nil {
				startCol, startRow = c, r
			}
		} else {
			if isHorizontal {
				startRow = 1
				startCol = nextColHorizontal
			} else {
				startRow = maxRow
				startCol = 1
			}
		}

		// Track section start for hiding
		sectionStartRow := startRow

		// Keep track of current row for this section
		currentRow := startRow

		// Handle title-only section
		if sectionType == SectionTypeTitleOnly {
			if sec.Title != "" {
				cell, _ := excelize.CoordinatesToCellName(startCol, currentRow)
				f.SetCellValue(sheet, cell, sec.Title)

				styleID, _ := createStyle(f, sec.TitleStyle)

				// Determine how many columns to merge for the title
				colSpan := sec.ColSpan
				if colSpan <= 1 && len(sec.Columns) > 1 {
					colSpan = len(sec.Columns)
				}

				if colSpan > 1 {
					endCell, _ := excelize.CoordinatesToCellName(startCol+colSpan-1, currentRow)
					f.MergeCell(sheet, cell, endCell)
					f.SetCellStyle(sheet, cell, endCell, styleID)
				} else {
					f.SetCellStyle(sheet, cell, cell, styleID)
				}

				currentRow++
			}

			// Update trackers and continue
			if currentRow > maxRow {
				maxRow = currentRow
			}

			// Adjust next column based on what was actually spanned
			colSpan := sec.ColSpan
			if colSpan <= 1 && len(sec.Columns) > 1 {
				colSpan = len(sec.Columns)
			}
			if colSpan <= 1 {
				colSpan = 1 // at least one column
			}
			nextColHorizontal = startCol + colSpan
			continue
		}

		// Render Title for full/hidden sections
		if sec.Title != "" {
			cell, _ := excelize.CoordinatesToCellName(startCol, currentRow)
			f.SetCellValue(sheet, cell, sec.Title)

			// Resolve style (Title defaults to bold)
			defaultTitle := &StyleTemplate{Font: &FontTemplate{Bold: true}}
			style := resolveStyle(sec.TitleStyle, defaultTitle, sec.Locked)
			styleID, _ := createStyle(f, style)

			// Merge title across columns if there are multiple columns
			if len(sec.Columns) > 1 {
				endCell, _ := excelize.CoordinatesToCellName(startCol+len(sec.Columns)-1, currentRow)
				f.MergeCell(sheet, cell, endCell)
				f.SetCellStyle(sheet, cell, endCell, styleID)
			} else {
				f.SetCellStyle(sheet, cell, cell, styleID)
			}

			currentRow++
		}

		// Render Hidden Field Name Row (if any column has HiddenFieldName)
		hasHiddenFields := false
		for _, col := range sec.Columns {
			if col.HiddenFieldName != "" {
				hasHiddenFields = true
				break
			}
		}

		if hasHiddenFields {
			// Style for hidden row (Yellow background, Locked)
			locked := true
			hiddenStyle := &StyleTemplate{
				Fill:   &FillTemplate{Color: "FFFF00"},
				Locked: &locked,
			}
			styleID, _ := createStyle(f, hiddenStyle)

			for i, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(startCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.HiddenFieldName)
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			hiddenRows = append(hiddenRows, currentRow)
			currentRow++
		}

		// Render Header
		if sec.ShowHeader {
			for i, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(startCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.Header)

				// Header uses column-specific locking
				locked := col.IsLocked(sec.Locked)

				// Default Header Style is Bold
				defaultHeader := &StyleTemplate{Font: &FontTemplate{Bold: true}}
				style := resolveStyle(sec.HeaderStyle, defaultHeader, locked)

				styleID, _ := createStyle(f, style)
				f.SetCellStyle(sheet, cell, cell, styleID)

				if col.Width > 0 {
					colName, _ := excelize.ColumnNumberToName(startCol + i)
					f.SetColWidth(sheet, colName, colName, col.Width)
				}
			}
			currentRow++
		}

		// Render Data
		dataVal := reflect.ValueOf(sec.Data)
		if dataVal.Kind() == reflect.Slice {
			for i := 0; i < dataVal.Len(); i++ {
				item := dataVal.Index(i)
				for j, col := range sec.Columns {
					val := extractValue(item, col.FieldName)

					// Apply formatter
					if col.Formatter != nil {
						val = col.Formatter(val)
					} else if col.FormatterName != "" {
						if fmtFunc, ok := e.formatters[col.FormatterName]; ok {
							val = fmtFunc(val)
						}
					}

					cell, _ := excelize.CoordinatesToCellName(startCol+j, currentRow)
					f.SetCellValue(sheet, cell, val)

					// Apply column-specific locking for data cells
					locked := col.IsLocked(sec.Locked)

					// Default Data Style is normal (nil), unless hidden section
					var defaultDataStyle *StyleTemplate
					if sectionType == SectionTypeHidden {
						defaultDataStyle = &StyleTemplate{
							Fill: &FillTemplate{Color: "FFFF00"},
						}
					}

					style := resolveStyle(sec.DataStyle, defaultDataStyle, locked)

					styleID, _ := createStyle(f, style)
					f.SetCellStyle(sheet, cell, cell, styleID)
				}
				currentRow++
			}
		}

		// Track hidden rows for hidden sections
		if sectionType == SectionTypeHidden {
			for r := sectionStartRow; r < currentRow; r++ {
				hiddenRows = append(hiddenRows, r)
			}
		}

		// Update global trackers
		if currentRow > maxRow {
			maxRow = currentRow
		}

		// Update next column for horizontal stacking
		nextColHorizontal = startCol + len(sec.Columns)
	}

	// Hide rows for hidden sections
	for _, r := range hiddenRows {
		f.SetRowVisible(sheet, r, false)
	}

	// Protect sheet if any cell needs locking
	if hasLockedCells {
		f.ProtectSheet(sheet, &excelize.SheetProtectionOptions{
			Password:            "",
			FormatCells:         false,
			FormatColumns:       true,
			FormatRows:          true,
			InsertColumns:       false,
			InsertRows:          false,
			InsertHyperlinks:    false,
			DeleteColumns:       false,
			DeleteRows:          false,
			Sort:                false,
			AutoFilter:          false,
			PivotTables:         false,
			SelectLockedCells:   true,
			SelectUnlockedCells: true,
		})
	}

	return nil
}

// resolveStyle merges defined style with default style and applies conditional locked styling.
func resolveStyle(base *StyleTemplate, defaultStyle *StyleTemplate, locked bool) *StyleTemplate {
	s := &StyleTemplate{}

	// Apply default if base is nil
	if base == nil {
		if defaultStyle != nil {
			*s = *defaultStyle
		}
	} else {
		*s = *base
		// If base has no font but default does, apply default font (rudimentary merge)
		if s.Font == nil && defaultStyle != nil && defaultStyle.Font != nil {
			s.Font = defaultStyle.Font
		}
		// If base has no fill but default does, apply default fill
		if s.Fill == nil && defaultStyle != nil && defaultStyle.Fill != nil {
			s.Fill = defaultStyle.Fill
		}
	}

	// Apply explicit lock override
	s.Locked = &locked

	// Auto-gray locked cells if no fill is explicitly set
	if locked && s.Fill == nil {
		s.Fill = &FillTemplate{Color: DefaultLockedColor}
	}

	return s
}

func extractValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Struct {
		f := item.FieldByName(fieldName)
		if f.IsValid() {
			return f.Interface()
		}
	} else if item.Kind() == reflect.Map {
		val := item.MapIndex(reflect.ValueOf(fieldName))
		if val.IsValid() {
			return val.Interface()
		}
	}
	return ""
}

func createStyle(f *excelize.File, tmpl *StyleTemplate) (int, error) {
	if tmpl == nil {
		return 0, nil
	}

	style := &excelize.Style{}
	if tmpl.Font != nil {
		style.Font = &excelize.Font{
			Bold:  tmpl.Font.Bold,
			Color: strings.TrimPrefix(tmpl.Font.Color, "#"),
		}
	}
	if tmpl.Fill != nil {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{strings.TrimPrefix(tmpl.Fill.Color, "#")},
			Pattern: 1,
		}
	}
	if tmpl.Locked != nil {
		style.Protection = &excelize.Protection{
			Locked: *tmpl.Locked,
		}
	}
	return f.NewStyle(style)
}

// mergeColumns merges user-defined columns with detected fields from data.
// It prioritizes user-defined columns, then appends remaining detected fields.
func mergeColumns(data interface{}, userConfigs []ColumnConfig) []ColumnConfig {
	if data == nil {
		return userConfigs
	}

	// 1. Detect all fields from data
	detectedFields := getFields(data)

	// 2. Index user configs by FieldName for O(1) lookup
	userConfigMap := make(map[string]ColumnConfig)
	seen := make(map[string]bool)
	var finalCols []ColumnConfig

	for _, col := range userConfigs {
		userConfigMap[col.FieldName] = col
		seen[col.FieldName] = true
		finalCols = append(finalCols, col)
	}

	// 3. Append detected fields that are not in user config
	for _, field := range detectedFields {
		if !seen[field] {
			// Create default config
			col := ColumnConfig{
				FieldName: field,
				Header:    field, // Default header is field name
				Width:     20,    // Default width
			}
			finalCols = append(finalCols, col)
			seen[field] = true
		}
	}

	return finalCols
}

func getFields(data interface{}) []string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// If not a slice, return empty (single item support could be added but usually export is slice)
	if v.Kind() != reflect.Slice {
		// Try to handle single struct if passed?
		// For now assume slice as per assumed usage, or standard usage.
		// If it's a single struct, we can treat it as one item.
		if v.Kind() == reflect.Struct {
			return getStructFields(v.Type())
		}
		return nil
	}

	if v.Len() == 0 {
		return nil
	}

	// Inspect first element
	elem := v.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	if elem.Kind() == reflect.Struct {
		return getStructFields(elem.Type())
	} else if elem.Kind() == reflect.Map {
		// Collect keys from all maps? Or just first?
		// Collecting from all is safer but slower.
		// For simplicity and performance, start with Union of first generic 10 rows?
		// Let's do union of all rows to be safe as maps can vary.
		// Limit to max 100 rows scan to prevent performance perf hit on large datasets?
		// Or just first row as convention?
		// "simpleexcel" implies simplicity. First row is standard convention for schema sniffing in basic libs.
		// BUT user said "no matter what... apply default".
		// To be robust, let's scan up to 10 rows.

		keysMap := make(map[string]bool)
		var keys []string

		limit := v.Len()
		if limit > 50 {
			limit = 50
		}

		for i := 0; i < limit; i++ {
			row := v.Index(i)
			if row.Kind() == reflect.Ptr {
				row = row.Elem()
			}
			if row.Kind() == reflect.Map {
				for _, key := range row.MapKeys() {
					k := key.String()
					if !keysMap[k] {
						keysMap[k] = true
						keys = append(keys, k)
					}
				}
			}
		}
		return keys
	}

	return nil
}

func getStructFields(t reflect.Type) []string {
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported
		if field.PkgPath != "" {
			continue
		}
		fields = append(fields, field.Name)
	}
	return fields
}
