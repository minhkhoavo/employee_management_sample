package simpleexcel

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
)

// DataExporter is the main entry point for exporting data.
type DataExporter struct {
	template *ReportTemplate
	// data holds data bound to specific section IDs (for YAML flow)
	data map[string]interface{}
	// sheets holds manually added sheets (for programmatic flow)
	sheets []*SheetBuilder
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
	FieldName string  `yaml:"field_name"` // Struct field name or map key
	Header    string  `yaml:"header"`
	Width     float64 `yaml:"width"`
	Locked    *bool   `yaml:"locked"` // Column-level lock override (overrides section Locked)
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
		data:   make(map[string]interface{}),
		sheets: []*SheetBuilder{},
	}
}

func NewDataExporterFromYamlFile(path string) (*DataExporter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open yaml file: %w", err)
	}
	defer f.Close()

	var tmpl ReportTemplate
	if err := yaml.NewDecoder(f).Decode(&tmpl); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return &DataExporter{
		template: &tmpl,
		data:     make(map[string]interface{}),
	}, nil
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

// BindDynamicSectionData binds data to a section ID and expands columns based on a map field.
func (e *DataExporter) BindDynamicSectionData(sectionID string, data interface{}, mapFieldName string) (*DataExporter, error) {
	// 1. Convert data
	dynamicData, newFields, err := ConvertStructsToDynamic(data, mapFieldName)
	if err != nil {
		return e, err
	}

	// 2. Bind converted data
	e.data[sectionID] = dynamicData

	// 3. Find section in template and expand columns
	if e.template != nil {
		found := false
		for i := range e.template.Sheets {
			for j := range e.template.Sheets[i].Sections {
				sec := &e.template.Sheets[i].Sections[j]
				if sec.ID == sectionID {
					sec.Columns = ExpandColumnConfigs(sec.Columns, mapFieldName, newFields)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return e, fmt.Errorf("section with ID %s not found in template", sectionID)
		}
	}

	return e, nil
}

// buildExcel creates an Excel file in memory and returns it
func (e *DataExporter) buildExcel() (*excelize.File, error) {
	f := excelize.NewFile()

	// 1. Process Programmatic Sheets
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			f.NewSheet(sheetName)
		}
		if err := e.renderSections(f, sheetName, sb.sections); err != nil {
			return nil, err
		}
	}

	// 2. Process YAML Template Sheets
	if e.template != nil {
		for i, sheetTmpl := range e.template.Sheets {
			sheetName := sheetTmpl.Name
			if len(e.sheets) == 0 && i == 0 {
				f.SetSheetName("Sheet1", sheetName)
			} else {
				idx, _ := f.GetSheetIndex(sheetName)
				if idx == -1 {
					f.NewSheet(sheetName)
				}
			}

			sections := make([]*SectionConfig, len(sheetTmpl.Sections))
			for j := range sheetTmpl.Sections {
				sec := &sheetTmpl.Sections[j]
				if data, ok := e.data[sec.ID]; ok {
					sec.Data = data
				}
				sections[j] = sec
			}

			if err := e.renderSections(f, sheetName, sections); err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

// hasComplexLayout checks if any section uses features that require in-memory processing,
// such as horizontal stacking or custom positioning.
func (e *DataExporter) hasComplexLayout() bool {
	// Check manually added sheets
	for _, sb := range e.sheets {
		for _, sec := range sb.sections {
			if sec.Direction == SectionDirectionHorizontal || sec.Position != "" {
				return true
			}
		}
	}

	// Check YAML template sheets
	if e.template != nil {
		for _, sheetTmpl := range e.template.Sheets {
			for _, sec := range sheetTmpl.Sections {
				if sec.Direction == SectionDirectionHorizontal || sec.Position != "" {
					return true
				}
			}
		}
	}

	return false
}

// StreamTo writes the Excel file to the provided writer using streaming.
// This is more memory efficient for large datasets.
// NOTE: If complex layout (e.g. horizontal sections) is detected, it falls back to
// in-memory usage (using buildExcel) to ensure correct rendering.
func (e *DataExporter) StreamTo(w io.Writer) error {
	// Fallback for complex layouts
	if e.hasComplexLayout() {
		f, err := e.buildExcel()
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteTo(w)
		return err
	}

	f := excelize.NewFile()
	defer f.Close()

	// Process Programmatic Sheets with streaming
	for i, sb := range e.sheets {
		sheetName := sb.name
		if i == 0 {
			f.SetSheetName("Sheet1", sheetName)
		} else {
			f.NewSheet(sheetName)
		}

		// Use streaming writer for the sheet
		sw, err := f.NewStreamWriter(sheetName)
		if err != nil {
			return fmt.Errorf("failed to create stream writer: %v", err)
		}

		// Render sections with streaming
		if err := e.streamSections(f, sw, sheetName, sb.sections); err != nil {
			return err
		}

		if err := sw.Flush(); err != nil {
			return fmt.Errorf("failed to flush stream: %v", err)
		}
	}

	// Process YAML Template Sheets with streaming
	if e.template != nil {
		for i, sheetTmpl := range e.template.Sheets {
			sheetName := sheetTmpl.Name
			if len(e.sheets) == 0 && i == 0 {
				f.SetSheetName("Sheet1", sheetName)
			} else {
				idx, _ := f.GetSheetIndex(sheetName)
				if idx == -1 {
					f.NewSheet(sheetName)
				}
			}

			sw, err := f.NewStreamWriter(sheetName)
			if err != nil {
				return fmt.Errorf("failed to create stream writer: %v", err)
			}

			sections := make([]*SectionConfig, len(sheetTmpl.Sections))
			for j := range sheetTmpl.Sections {
				sec := &sheetTmpl.Sections[j]
				if data, ok := e.data[sec.ID]; ok {
					sec.Data = data
				}
				sections[j] = sec
			}

			if err := e.streamSections(f, sw, sheetName, sections); err != nil {
				return err
			}

			if err := sw.Flush(); err != nil {
				return fmt.Errorf("failed to flush stream: %v", err)
			}
		}
	}

	// Write the file to the provided writer
	_, err := f.WriteTo(w)
	return err
}

// streamSections renders sections using streaming writer
func (e *DataExporter) streamSections(f *excelize.File, sw *excelize.StreamWriter, sheet string, sections []*SectionConfig) error {
	rowNum := 1
	hiddenRows := []int{}   // Track rows to hide
	merges := [][2]string{} // Track cells to merge [start, end]

	for _, sec := range sections {
		sectionStartRow := rowNum

		// Determine section type
		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeFull
		}

		// Handle title-only section
		if sectionType == SectionTypeTitleOnly {
			if sec.Title != "" {
				style, _ := createStyle(f, sec.TitleStyle)
				header := []interface{}{sec.Title}
				cell, _ := excelize.CoordinatesToCellName(1, rowNum)
				sw.SetRow(cell, header, excelize.RowOpts{StyleID: style})

				// If ColSpan is specified, record the merge to be applied later
				if sec.ColSpan > 1 {
					endCell, _ := excelize.CoordinatesToCellName(sec.ColSpan, rowNum)
					merges = append(merges, [2]string{cell, endCell})
				}

				rowNum++
			}
			// Add spacing
			rowNum++
			continue
		}

		// Handle full and hidden sections
		if sec.Title != "" {
			style, _ := createStyle(f, sec.TitleStyle)
			header := []interface{}{sec.Title}
			cell, _ := excelize.CoordinatesToCellName(1, rowNum)
			sw.SetRow(cell, header, excelize.RowOpts{StyleID: style})

			// If there are multiple columns, record the merge
			if len(sec.Columns) > 1 {
				endCell, _ := excelize.CoordinatesToCellName(len(sec.Columns), rowNum)
				merges = append(merges, [2]string{cell, endCell})
			}
			rowNum++
		}

		if sec.ShowHeader && len(sec.Columns) > 0 {
			headers := make([]interface{}, len(sec.Columns))
			for i, col := range sec.Columns {
				headers[i] = col.Header
			}
			cell, _ := excelize.CoordinatesToCellName(1, rowNum)
			sw.SetRow(cell, headers)
			rowNum++
		}

		// Process data rows
		if sec.Data != nil {
			v := reflect.ValueOf(sec.Data)
			if v.Kind() == reflect.Slice && v.Len() > 0 {
				for i := 0; i < v.Len(); i++ {
					item := v.Index(i).Interface()
					row := make([]interface{}, len(sec.Columns))
					for j, col := range sec.Columns {
						row[j] = extractValue(reflect.ValueOf(item), col.FieldName)
					}
					cell, _ := excelize.CoordinatesToCellName(1, rowNum)
					if err := sw.SetRow(cell, row); err != nil {
						return fmt.Errorf("error writing row %d: %v", i+1, err)
					}
					rowNum++

					// Flush every 1000 rows to manage memory
					if rowNum%1000 == 0 {
						if err := sw.Flush(); err != nil {
							return fmt.Errorf("error flushing rows: %v", err)
						}
					}
				}
			}
		}

		// Track hidden rows for hidden sections
		if sectionType == SectionTypeHidden {
			for r := sectionStartRow; r < rowNum; r++ {
				hiddenRows = append(hiddenRows, r)
			}
		}

		// Add spacing between sections
		rowNum++
	}

	// After streaming is done, apply merges and hide rows
	if err := sw.Flush(); err != nil {
		return fmt.Errorf("error flushing before applying styles: %v", err)
	}

	for _, m := range merges {
		f.MergeCell(sheet, m[0], m[1])
	}

	for _, r := range hiddenRows {
		f.SetRowVisible(sheet, r, false)
	}

	return nil
}

// ExportToExcel generates the Excel file on disk.
func (e *DataExporter) ExportToExcel(ctx context.Context, path string) error {
	f, err := e.buildExcel()
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SaveAs(path)
}

// ToBytes exports the Excel file to an in-memory byte slice.
func (e *DataExporter) ToBytes() ([]byte, error) {
	f, err := e.buildExcel()
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

// ToWriter writes the Excel file to the provided io.Writer.
// For better memory efficiency with large datasets, use StreamTo instead.
func (e *DataExporter) ToWriter(w io.Writer) error {
	// Use streaming for better memory efficiency
	return e.StreamTo(w)
}

// StreamToResponse writes the Excel file directly to an HTTP response writer.
// This is useful for streaming large Excel files in web handlers.
func (e *DataExporter) StreamToResponse(w http.ResponseWriter, filename string) error {
	// Set headers
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Transfer-Encoding", "binary")

	// Stream the file directly to the response
	return e.StreamTo(w)
}

// ToCSV exports the first sheet as CSV to the provided io.Writer.
// This implementation is memory efficient as it streams data directly
// without loading the entire Excel file into memory.
func (e *DataExporter) ToCSV(w io.Writer) error {
	// Create a temporary file for the Excel data
	tmpFile, err := os.CreateTemp("", "excel-*.xlsx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write Excel data to temp file
	if err := e.StreamTo(tmpFile); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Open the Excel file for reading
	f, err := excelize.OpenFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %v", err)
	}
	defer f.Close()

	// Get the first sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found")
	}
	sheet := sheets[0]

	// Create a CSV writer
	csvWriter := csv.NewWriter(w)

	// Stream rows directly from the Excel file
	rows, err := f.Rows(sheet)
	if err != nil {
		return fmt.Errorf("failed to get rows: %v", err)
	}

	// Read and write rows in chunks
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("error reading row: %v", err)
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("error writing CSV row: %v", err)
		}

		// Flush periodically to manage memory
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("error flushing CSV: %v", err)
		}
	}

	// Check for iteration errors
	if err = rows.Close(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	// Flush any remaining data
	csvWriter.Flush()
	return csvWriter.Error()
}

// ToCSVBytes exports the first sheet as CSV and returns it as a byte slice.
func (e *DataExporter) ToCSVBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := e.ToCSV(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ToJSON exports the data as JSON to the provided io.Writer.
// Only works with structured data (slices of structs or maps).
func (e *DataExporter) ToJSON(w io.Writer) error {
	var data []map[string]interface{}

	// Get data from the first section with data
	for _, sb := range e.sheets {
		for _, section := range sb.sections {
			if section.Data != nil {
				// Convert the data to a slice of maps
				rv := reflect.ValueOf(section.Data)
				if rv.Kind() == reflect.Slice && rv.Len() > 0 {
					data = make([]map[string]interface{}, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						item := rv.Index(i)
						if item.Kind() == reflect.Ptr {
							item = item.Elem()
						}

						if item.Kind() == reflect.Struct {
							m := make(map[string]interface{})
							t := item.Type()
							for j := 0; j < t.NumField(); j++ {
								field := t.Field(j)
								// Skip unexported fields
								if field.PkgPath != "" {
									continue
								}
								m[field.Name] = item.Field(j).Interface()
							}
							data[i] = m
						}
					}
				}
				break
			}
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// ToJSONString exports the data as a JSON string.
func (e *DataExporter) ToJSONString() (string, error) {
	var buf bytes.Buffer
	if err := e.ToJSON(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
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
			// Check if any column is explicitly locked
			for _, col := range sec.Columns {
				if col.Locked != nil && *col.Locked {
					hasLockedCells = true
					break
				}
			}
		}

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

			style := getEffectiveStyle(sec.TitleStyle, sec.Locked, true)
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

		// Render Header
		if sec.ShowHeader {
			for i, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(startCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.Header)

				// Header uses column-specific locking
				locked := col.IsLocked(sec.Locked)
				style := getEffectiveStyle(sec.HeaderStyle, locked, true)
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
					cell, _ := excelize.CoordinatesToCellName(startCol+j, currentRow)
					f.SetCellValue(sheet, cell, val)

					// Apply column-specific locking for data cells
					locked := col.IsLocked(sec.Locked)
					style := getEffectiveStyle(sec.DataStyle, locked, false)
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
			FormatColumns:       false,
			FormatRows:          false,
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

// getEffectiveStyle returns a style with the appropriate lock setting.
// locked parameter determines if this cell should be locked.
func getEffectiveStyle(base *StyleTemplate, locked bool, isHeaderOrTitle bool) *StyleTemplate {
	s := &StyleTemplate{}
	if base != nil {
		*s = *base
	}
	s.Locked = &locked
	return s
}

func extractValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Struct {
		f := item.FieldByName(fieldName)
		if f.IsValid() {
			return f.Interface()
		}
	}
	// Handle maps if needed, but struct is primary use case
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
