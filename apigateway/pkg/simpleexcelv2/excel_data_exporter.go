package simpleexcelv2

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
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

// ExcelDataExporter is the main entry point for exporting data.
type ExcelDataExporter struct {
	template *ReportTemplate
	// data holds data bound to specific section IDs (for YAML flow)
	data map[string]interface{}
	// sheets holds manually added sheets (for programmatic flow)
	sheets []*SheetBuilder
	// formatters holds registered formatter functions by name
	formatters map[string]func(interface{}) interface{}

	// Metadata for coordinate mapping
	sectionMetadata map[string]SectionPlacement
}

// SectionPlacement stores the starting coordinates and metadata of a rendered section.
type SectionPlacement struct {
	SectionID    string
	StartRow     int
	StartCol     int
	FieldOffsets map[string]int // Map of FieldName to ColumnOffset (relative to startCol)
	DataLen      int            // Number of data rows
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
	ID             string         `yaml:"id"`
	Title          string         `yaml:"title"`
	ColSpan        int            `yaml:"col_span"`        // Number of columns to span for title-only sections
	Data           interface{}    `yaml:"-"`               // Data is bound at runtime
	SourceSections []string       `yaml:"source_sections"` // IDs of sections this depends on
	Type           string         `yaml:"type"`            // "full", "title", "hidden"
	Locked         bool           `yaml:"locked"`          // Section-level lock (default for all columns)
	ShowHeader     bool           `yaml:"show_header"`
	Direction      string         `yaml:"direction"` // "horizontal" or "vertical"
	Position       string         `yaml:"position"`  // e.g., "A1"
	TitleStyle     *StyleTemplate `yaml:"title_style"`
	HeaderStyle    *StyleTemplate `yaml:"header_style"`
	DataStyle      *StyleTemplate `yaml:"data_style"`
	TitleHeight    float64        `yaml:"title_height"`
	HeaderHeight   float64        `yaml:"header_height"`
	DataHeight     float64        `yaml:"data_height"`
	HasFilter      bool           `yaml:"has_filter"`
	Columns        []ColumnConfig `yaml:"columns"`
}

// CompareConfig defines how to compare a column with another section.
type CompareConfig struct {
	SectionID string `yaml:"section_id"`
	FieldName string `yaml:"field_name"`
}

// ColumnConfig defines a column in a section.
type ColumnConfig struct {
	FieldName       string                        `yaml:"field_name"` // Struct field name or map key
	Header          string                        `yaml:"header"`
	Width           float64                       `yaml:"width"`
	Height          float64                       `yaml:"height"`
	Locked          *bool                         `yaml:"locked"`            // Column-level lock override (overrides section Locked)
	Formatter       func(interface{}) interface{} `yaml:"-"`                 // Optional custom formatter function (Programmatic)
	FormatterName   string                        `yaml:"formatter"`         // Name of registered formatter (YAML)
	HiddenFieldName string                        `yaml:"hidden_field_name"` // Hidden field name for backend use
	CompareWith     *CompareConfig                `yaml:"compare_with"`      // For injecting comparison formulas
	CompareAgainst  *CompareConfig                `yaml:"compare_against"`   // For injecting comparison formulas
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
	Font      *FontTemplate      `yaml:"font"`
	Fill      *FillTemplate      `yaml:"fill"`
	Alignment *AlignmentTemplate `yaml:"alignment"`
	Locked    *bool              `yaml:"locked"`
}

type AlignmentTemplate struct {
	Horizontal string `yaml:"horizontal"` // center, left, right
	Vertical   string `yaml:"vertical"`   // top, center, bottom
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

func NewExcelDataExporter() *ExcelDataExporter {
	return &ExcelDataExporter{
		data:            make(map[string]interface{}),
		sheets:          []*SheetBuilder{},
		formatters:      make(map[string]func(interface{}) interface{}),
		sectionMetadata: make(map[string]SectionPlacement),
	}
}

func NewExcelDataExporterFromYamlConfig(yamlConfig string) (*ExcelDataExporter, error) {
	var tmpl ReportTemplate
	if yamlConfig == "" {
		return nil, fmt.Errorf("yaml config is empty")
	}
	if err := yaml.Unmarshal([]byte(yamlConfig), &tmpl); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	exporter := &ExcelDataExporter{
		template:        &tmpl,
		data:            make(map[string]interface{}),
		formatters:      make(map[string]func(interface{}) interface{}),
		sheets:          make([]*SheetBuilder, 0),
		sectionMetadata: make(map[string]SectionPlacement),
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
func (e *ExcelDataExporter) AddSheet(name string) *SheetBuilder {
	sb := &SheetBuilder{
		exporter: e,
		name:     name,
		sections: []*SectionConfig{},
	}
	e.sheets = append(e.sheets, sb)
	return sb
}

// BindSectionData binds data to a section ID (for YAML-based export).
func (e *ExcelDataExporter) BindSectionData(id string, data interface{}) *ExcelDataExporter {
	e.data[id] = data
	return e
}

// RegisterFormatter registers a formatter function with a name.
// This allows referencing formatters by name in YAML configurations.
func (e *ExcelDataExporter) RegisterFormatter(name string, f func(interface{}) interface{}) *ExcelDataExporter {
	e.formatters[name] = f
	return e
}

// GetSheet returns a SheetBuilder by name, or nil if not found.
func (e *ExcelDataExporter) GetSheet(name string) *SheetBuilder {
	for _, sheet := range e.sheets {
		if sheet.name == name {
			return sheet
		}
	}
	return nil
}

// GetSheetByIndex returns a SheetBuilder by index (0-based), or nil if out of bounds.
func (e *ExcelDataExporter) GetSheetByIndex(index int) *SheetBuilder {
	if index < 0 || index >= len(e.sheets) {
		return nil
	}
	return e.sheets[index]
}

// BuildExcel constructs an Excel file (*excelize.File) based on the exporter's configuration and data.
// It processes both programmatically added sheets and sheets defined in a YAML template,
// returning the generated excelize.File instance or an error// BuildExcel generates the excel file
func (e *ExcelDataExporter) BuildExcel() (*excelize.File, error) {
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
func (e *ExcelDataExporter) ExportToExcel(ctx context.Context, path string) error {
	f, err := e.BuildExcel()
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SaveAs(path)
}

// ToBytes exports the Excel file to an in-memory byte slice.
func (e *ExcelDataExporter) ToBytes() ([]byte, error) {
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

// ToWriter exports the Excel file directly to a writer.
func (e *ExcelDataExporter) ToWriter(w io.Writer) error {
	f, err := e.BuildExcel()
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Write(w)
}

// ToCSV exports the first sheet of data to CSV format.
// This is significantly more memory-efficient for very large datasets as it avoids Excel overhead.
func (e *ExcelDataExporter) ToCSV(w io.Writer) error {
	if len(e.sheets) == 0 {
		return fmt.Errorf("no sheets to export")
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	sheet := e.sheets[0]
	for _, sec := range sheet.sections {
		// Perform Late Binding if needed
		if sec.ID != "" && sec.Data == nil {
			if data, ok := e.data[sec.ID]; ok {
				sec.Data = data
			}
		}

		// Get data length
		dataLen := e.getDataLength(sec)
		if dataLen == 0 && !sec.ShowHeader {
			continue
		}

		// Resolve columns
		cols := mergeColumns(sec.Data, sec.Columns)

		// Title (if single title only)
		if sec.Title != "" {
			_ = csvWriter.Write([]string{sec.Title})
		}

		// Header
		if sec.ShowHeader && len(cols) > 0 {
			headerArr := make([]string, len(cols))
			for i, col := range cols {
				headerArr[i] = col.Header
			}
			if err := csvWriter.Write(headerArr); err != nil {
				return err
			}
		}

		// Data
		if dataLen > 0 {
			v := reflect.ValueOf(sec.Data)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			for i := 0; i < dataLen; i++ {
				item := v.Index(i)
				rowArr := make([]string, len(cols))
				for j, col := range cols {
					val := extractValue(item, col.FieldName)
					// Apply formatter if any
					if col.Formatter != nil {
						val = col.Formatter(val)
					} else if col.FormatterName != "" && e.formatters != nil {
						if fn, ok := e.formatters[col.FormatterName]; ok {
							val = fn(val)
						}
					}
					rowArr[j] = fmt.Sprintf("%v", val)
				}
				if err := csvWriter.Write(rowArr); err != nil {
					return err
				}
			}
		}

		// Empty line between sections
		_ = csvWriter.Write([]string{""})
	}

	return nil
}

// =============================================================================
// SheetBuilder
// =============================================================================

type SheetBuilder struct {
	exporter *ExcelDataExporter
	name     string
	sections []*SectionConfig
}

func (sb *SheetBuilder) AddSection(config *SectionConfig) *SheetBuilder {
	sb.sections = append(sb.sections, config)
	return sb
}

func (sb *SheetBuilder) Build() *ExcelDataExporter {
	return sb.exporter
}

// =============================================================================
// Rendering Logic
// =============================================================================

// hasHiddenFields returns true if any column in the section has a HiddenFieldName.
func hasHiddenFields(sec *SectionConfig) bool {
	for _, col := range sec.Columns {
		if col.HiddenFieldName != "" {
			return true
		}
	}
	return false
}

// calculatePosition returns the start coordinates for a section.
func calculatePosition(sec *SectionConfig, nextColHorizontal, maxRow int) (int, int) {
	if sec.Position != "" {
		c, r, err := excelize.CellNameToCoordinates(sec.Position)
		if err == nil {
			return c, r
		}
	}

	isHorizontal := sec.Direction == SectionDirectionHorizontal
	if isHorizontal {
		return nextColHorizontal, 1
	}
	return 1, maxRow
}

// getDataLength returns the expected number of data rows for a section.
func (e *ExcelDataExporter) getDataLength(sec *SectionConfig) int {
	dataVal := reflect.ValueOf(sec.Data)
	if dataVal.Kind() == reflect.Slice {
		return dataVal.Len()
	}
	if len(sec.SourceSections) > 0 {
		if sourcePlacement, ok := e.sectionMetadata[sec.SourceSections[0]]; ok {
			return sourcePlacement.DataLen
		}
	}
	return 0
}

func (e *ExcelDataExporter) renderSections(f *excelize.File, sheet string, sections []*SectionConfig) error {
	// --- PASS 1: Layout Calculation ---
	tempRow, tempCol := 1, 1
	maxRowForPass1 := 1

	placements := make([]SectionPlacement, len(sections))

	for i, sec := range sections {
		// Determine section type
		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeFull
		}

		// Determine effective columns merging user config and data fields
		sec.Columns = mergeColumns(sec.Data, sec.Columns)

		// Determine start coordinates
		sCol, sRow := calculatePosition(sec, tempCol, tempRow)

		// Calculate data start row by skipping Title, Hidden Row, and Header
		dataStartRow := sRow
		if sectionType != SectionTypeTitleOnly {
			if sec.Title != "" {
				dataStartRow++
			}
			if hasHiddenFields(sec) {
				dataStartRow++
			}
			if sec.ShowHeader {
				dataStartRow++
			}
		} else {
			if sec.Title != "" {
				dataStartRow++
			}
		}

		fieldOffsets := make(map[string]int)
		for j, col := range sec.Columns {
			fieldOffsets[col.FieldName] = j
		}

		// We need to know DataLen for Pass 1 to update tempRow/tempCol trackers accurately
		dataLen := e.getDataLength(sec)

		placements[i] = SectionPlacement{
			SectionID:    sec.ID,
			StartRow:     dataStartRow,
			StartCol:     sCol,
			FieldOffsets: fieldOffsets,
			DataLen:      dataLen,
		}

		if sec.ID != "" {
			e.sectionMetadata[sec.ID] = placements[i]
		}

		// Update global trackers for Pass 1 layout
		finishRow := dataStartRow + dataLen
		if finishRow > maxRowForPass1 {
			maxRowForPass1 = finishRow
		}
		if finishRow > tempRow {
			tempRow = finishRow // This is for vertical stacking logic if we were purely vertical
		}

		// For horizontal tracking
		colSpan := len(sec.Columns)
		if sectionType == SectionTypeTitleOnly {
			colSpan = sec.ColSpan
			if colSpan <= 1 && len(sec.Columns) > 1 {
				colSpan = len(sec.Columns)
			}
		}
		tempCol = sCol + colSpan
	}

	// --- PASS 2: Actual Rendering ---
	maxRow := 1
	nextColHorizontal := 1
	hasLockedCells := false
	hiddenRows := []int{}

	// Check for locked cells first (to decide if we need to unlock sheet)
	for _, sec := range sections {
		if sec.Locked {
			hasLockedCells = true
		} else {
			for _, col := range sec.Columns {
				if col.Locked != nil && *col.Locked {
					hasLockedCells = true
					break
				}
			}
		}
		if hasLockedCells {
			break
		}
	}

	if hasLockedCells {
		unlocked := false
		defaultStyle := &StyleTemplate{Locked: &unlocked}
		styleID, _ := createStyle(f, defaultStyle)
		f.SetColStyle(sheet, "A:XFD", styleID)
	}

	for i, sec := range sections {
		placement := placements[i]

		// Re-calculate sCol, sRow for Pass 2 (should match Pass 1)
		sCol, sRow := calculatePosition(sec, nextColHorizontal, maxRow)
		currentRow := sRow

		sectionType := sec.Type
		if sectionType == "" {
			sectionType = SectionTypeFull
		}

		// Handle Title Only
		if sectionType == SectionTypeTitleOnly {
			if sec.Title != "" {
				cell, _ := excelize.CoordinatesToCellName(sCol, currentRow)
				f.SetCellValue(sheet, cell, sec.Title)
				defaultTitleOnly := &StyleTemplate{
					Font:      &FontTemplate{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				style := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
				styleID, _ := createStyle(f, style)
				colSpan := sec.ColSpan
				if colSpan <= 1 && len(sec.Columns) > 1 {
					colSpan = len(sec.Columns)
				}
				if colSpan > 1 {
					endCell, _ := excelize.CoordinatesToCellName(sCol+colSpan-1, currentRow)
					f.MergeCell(sheet, cell, endCell)
					f.SetCellStyle(sheet, cell, endCell, styleID)
				} else {
					f.SetCellStyle(sheet, cell, cell, styleID)
				}
				if sec.TitleHeight > 0 {
					f.SetRowHeight(sheet, currentRow, sec.TitleHeight)
				}
				currentRow++
			}
			if currentRow > maxRow {
				maxRow = currentRow
			}
			colSpan := sec.ColSpan
			if colSpan <= 1 && len(sec.Columns) > 1 {
				colSpan = len(sec.Columns)
			}
			if colSpan <= 1 {
				colSpan = 1
			}
			nextColHorizontal = sCol + colSpan
			continue
		}

		// Render Title
		if sec.Title != "" {
			cell, _ := excelize.CoordinatesToCellName(sCol, currentRow)
			f.SetCellValue(sheet, cell, sec.Title)
			defaultTitle := &StyleTemplate{
				Font:      &FontTemplate{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			style := resolveStyle(sec.TitleStyle, defaultTitle, sec.Locked)
			styleID, _ := createStyle(f, style)
			if len(sec.Columns) > 1 {
				endCell, _ := excelize.CoordinatesToCellName(sCol+len(sec.Columns)-1, currentRow)
				f.MergeCell(sheet, cell, endCell)
				f.SetCellStyle(sheet, cell, endCell, styleID)
			} else {
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			if sec.TitleHeight > 0 {
				f.SetRowHeight(sheet, currentRow, sec.TitleHeight)
			}
			currentRow++
		}

		// Render Hidden Field Name Row
		if hasHiddenFields(sec) {
			locked := true
			hiddenStyle := &StyleTemplate{Fill: &FillTemplate{Color: "FFFF00"}, Locked: &locked}
			styleID, _ := createStyle(f, hiddenStyle)
			for i, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(sCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.HiddenFieldName)
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			hiddenRows = append(hiddenRows, currentRow)
			currentRow++
		}

		// Render Header
		if sec.ShowHeader {
			for i, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(sCol+i, currentRow)
				f.SetCellValue(sheet, cell, col.Header)
				locked := col.IsLocked(sec.Locked)
				defaultHeader := &StyleTemplate{
					Font:      &FontTemplate{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				style := resolveStyle(sec.HeaderStyle, defaultHeader, locked)
				styleID, _ := createStyle(f, style)
				f.SetCellStyle(sheet, cell, cell, styleID)
				if col.Width > 0 {
					colName, _ := excelize.ColumnNumberToName(sCol + i)
					f.SetColWidth(sheet, colName, colName, col.Width)
				}
			}
			if sec.HeaderHeight > 0 {
				f.SetRowHeight(sheet, currentRow, sec.HeaderHeight)
			}
			currentRow++
		}

		// Render Data
		dataLen := placement.DataLen // Use pre-calculated length
		dataVal := reflect.ValueOf(sec.Data)
		for i := 0; i < dataLen; i++ {
			var item reflect.Value
			if dataVal.Kind() == reflect.Slice && i < dataVal.Len() {
				item = dataVal.Index(i)
			}
			for j, col := range sec.Columns {
				cell, _ := excelize.CoordinatesToCellName(sCol+j, currentRow)
				if col.CompareWith != nil {
					formula, err := e.generateDiffFormula(col, i)
					if err == nil {
						f.SetCellFormula(sheet, cell, formula)
					} else {
						f.SetCellValue(sheet, cell, fmt.Sprintf("Error: %v", err))
					}
				} else if item.IsValid() {
					val := extractValue(item, col.FieldName)
					if col.Formatter != nil {
						val = col.Formatter(val)
					} else if col.FormatterName != "" {
						if fmtFunc, ok := e.formatters[col.FormatterName]; ok {
							val = fmtFunc(val)
						}
					}
					f.SetCellValue(sheet, cell, val)
				}

				locked := col.IsLocked(sec.Locked)
				var defaultDataStyle *StyleTemplate
				if sectionType == SectionTypeHidden {
					defaultDataStyle = &StyleTemplate{Fill: &FillTemplate{Color: "FFFF00"}}
				}
				style := resolveStyle(sec.DataStyle, defaultDataStyle, locked)
				styleID, _ := createStyle(f, style)
				f.SetCellStyle(sheet, cell, cell, styleID)
			}
			// Apply data row height
			rowHeight := sec.DataHeight
			for _, col := range sec.Columns {
				if col.Height > rowHeight {
					rowHeight = col.Height
				}
			}
			if rowHeight > 0 {
				f.SetRowHeight(sheet, currentRow, rowHeight)
			}
			currentRow++
		}

		// Apply AutoFilter if requested
		if sec.HasFilter && sec.ShowHeader && len(sec.Columns) > 0 {
			headerRow := sRow
			if sec.Title != "" {
				headerRow++
			}
			if hasHiddenFields(sec) {
				headerRow++
			}
			// headerRow is now the row index of the header

			firstCell, _ := excelize.CoordinatesToCellName(sCol, headerRow)
			lastCell, _ := excelize.CoordinatesToCellName(sCol+len(sec.Columns)-1, currentRow-1)
			filterRange := fmt.Sprintf("%s:%s", firstCell, lastCell)
			f.AutoFilter(sheet, filterRange, nil)
		}

		if sectionType == SectionTypeHidden {
			for r := sRow; r < currentRow; r++ {
				hiddenRows = append(hiddenRows, r)
			}
		}

		if currentRow > maxRow {
			maxRow = currentRow
		}
		nextColHorizontal = sCol + len(sec.Columns)
	}

	for _, r := range hiddenRows {
		f.SetRowVisible(sheet, r, false)
	}

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
			AutoFilter:          true,
			PivotTables:         false,
			SelectLockedCells:   true,
			SelectUnlockedCells: true,
		})
	}

	return nil
}

func (e *ExcelDataExporter) resolveCellAddress(sectionID, fieldName string, rowOffset int) (string, error) {
	placement, ok := e.sectionMetadata[sectionID]
	if !ok {
		return "", fmt.Errorf("section %s not found", sectionID)
	}

	colOffset, ok := placement.FieldOffsets[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in %s", fieldName, sectionID)
	}

	// StartRow in metadata should point to the first row of DATA
	return excelize.CoordinatesToCellName(placement.StartCol+colOffset, placement.StartRow+rowOffset)
}

func (e *ExcelDataExporter) generateDiffFormula(col ColumnConfig, rowOffset int) (string, error) {
	if col.CompareWith == nil {
		return "", nil
	}

	cellA, err := e.resolveCellAddress(col.CompareWith.SectionID, col.CompareWith.FieldName, rowOffset)
	if err != nil {
		return "", err
	}

	if col.CompareAgainst != nil {
		cellB, err := e.resolveCellAddress(col.CompareAgainst.SectionID, col.CompareAgainst.FieldName, rowOffset)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`IF(%s<>%s, "Diff", "")`, cellA, cellB), nil
	}

	// Default comparison is not specified in the plan but let's assume it compares with something else if CompareAgainst is nil?
	// The plan says: =IF(Editable_Cell <> Original_Cell, "Diff", "")
	// If only CompareWith is provided, maybe it's compared against the current section's field?
	// Let's re-read the plan.
	// Plan says:
	// cellA, _ := e.resolveCellAddress(col.CompareWith.SectionID, col.CompareWith.FieldName, i)
	// cellB, _ := e.resolveCellAddress(col.CompareAgainst.SectionID, col.CompareAgainst.FieldName, i)
	// formula := fmt.Sprintf(`IF(%s<>%s, "Diff", "")`, cellA, cellB)

	// If CompareAgainst is nil, we should return an error or handle it.
	return "", fmt.Errorf("CompareAgainst is required for comparison column %s", col.FieldName)
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
		// If base has no alignment but default does, apply default alignment
		if s.Alignment == nil && defaultStyle != nil && defaultStyle.Alignment != nil {
			s.Alignment = defaultStyle.Alignment
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
	if tmpl.Alignment != nil {
		style.Alignment = &excelize.Alignment{
			Horizontal: tmpl.Alignment.Horizontal,
			Vertical:   tmpl.Alignment.Vertical,
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
