# pgexcel - Data Exporter

A Go library for exporting in-memory data structures to Excel files with support for templates, styling, protection, and flexible layouts.

## Overview

This package exports Go slices (of structs or maps) to Excel (.xlsx) files using the [excelize](https://github.com/xuri/excelize) library. It supports:

- **Struct/Map to Excel**: Automatic column detection via reflection
- **YAML Templates**: Define styling, layout, and protection via YAML
- **YAML Section Config**: Define sections in YAML with runtime data binding
- **Stacked Sections**: Multiple data collections on a single sheet (vertical or horizontal)
- **Per-Section Styling**: Different colors, fonts, and formats per section
- **Cell Protection**: Lock/unlock sections for editing control

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        data_exporter.go                         │
├─────────────────────────────────────────────────────────────────┤
│  Core Types          │  Template Types      │  Export Logic     │
│  ├── CellStyle       │  ├── ReportTemplate  │  ├── DataExporter │
│  ├── SheetProtection │  ├── SheetTemplate   │  ├── SheetBuilder │
│  ├── CellRange       │  ├── ColumnTemplate  │  ├── Export()     │
│  └── RowRange        │  ├── StyleTemplate   │  ├── exportSheet()│
│                      │  ├── LayoutTemplate  │  └── exportSections()
│  Section Types       │  └── ProtectionTmpl  │                   │
│  ├── SectionConfig   │                      │  Utilities        │
│  └── ColumnConfig    │  Template Loading    │  └── columnIndexToName()
│                      │  ├── LoadTemplate()  │                   │
│                      │  └── ValidateTemplate()                  │
└─────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Self-Contained File
All types, template loading, and export logic are in a single file (`data_exporter.go`) for simplicity. This was a deliberate consolidation from multiple files.

### 2. Fluent API Pattern
```go
NewDataExporter().
    AddSheet("Sheet1").
        AddSection(&SectionConfig{...}).
        AddSection(&SectionConfig{...}).
        Build().
    ExportToFile(ctx, "output.xlsx")
```

### 3. Three Export Modes
- **Single Data Mode**: `WithData(sheetName, slice)` - one slice per sheet
- **Section Mode (Programmatic)**: `AddSection()` - multiple collections stacked on one sheet
- **Section Mode (YAML)**: `BindSectionData()` - sections defined in YAML with data bound at runtime

### 4. Direction-Aware Layout
Sections can stack:
- **Vertically** (default): rows below each other
- **Horizontally**: columns side-by-side

## Public API

### Constructors

| Function | Description |
|----------|-------------|
| `NewDataExporter()` | Create exporter without template |
| `NewDataExporterWithTemplate(template)` | Create with parsed template |
| `NewDataExporterFromYaml(yamlContent)` | Create from YAML string |
| `NewDataExporterFromYamlFile(path)` | Load template from YAML file |

### DataExporter Methods

| Method | Description |
|--------|-------------|
| `WithData(sheetName, data)` | Add slice data for a sheet |
| `AddSheet(sheetName)` | Start building a sheet (returns SheetBuilder) |
| `BindSectionData(sectionID, data)` | Bind data to a YAML-defined section by ID |
| `Export(ctx, writer)` | Write Excel to io.Writer |
| `ExportToFile(ctx, path)` | Write Excel to file |

### SheetBuilder Methods

| Method | Description |
|--------|-------------|
| `WithData(data)` | Set single data slice |
| `AddSection(config)` | Add a section (supports stacking) |
| `WithLayout(layout)` | Set layout options |
| `WithProtection(protection)` | Set sheet protection |
| `Build()` | Finalize and return to DataExporter |

## SectionConfig Fields

```go
type SectionConfig struct {
    // Identification (for YAML binding)
    ID          string          // Section ID for BindSectionData() matching

    // Content
    Title       string          // Optional title row
    Data        interface{}     // Slice of structs or maps (runtime only)
    ShowHeader  bool            // Show column headers
    
    // Protection
    Locked      bool            // Lock this section from editing
    
    // Layout
    GapAfter    int             // Gap after section: rows (vertical) or columns (horizontal)
    Direction   string          // "vertical" (default) or "horizontal"
    Position    string          // Excel-style position (e.g., "A1", "B3"). Overrides StartColumn/StartRow if both are set
    StartColumn int             // Starting column (0-based) for horizontal (alternative to Position)
    StartRow    int             // Starting row (1-based) for horizontal (alternative to Position)
    
    // Styling
    TitleStyle  *StyleTemplate  // Title row style
    HeaderStyle *StyleTemplate  // Header row style
    DataStyle   *StyleTemplate  // Data cells style
    
    // Column Overrides
    Columns     []ColumnConfig  // Override headers, widths, formats
}
```

## YAML Section Configuration

### Excel-Style Positioning

You can position sections using Excel-style cell references (e.g., "A1", "B3") for more intuitive placement:

```yaml
sections:
  - id: "summary"
    title: "Summary"
    position: "A1"  # Starts at cell A1
    # ... other settings

  - id: "details"
    title: "Details"
    position: "D5"  # Starts at cell D5
    # ... other settings
```

Or use it programmatically:

```go
AddSection(&SectionConfig{
    ID:       "summary",
    Title:    "Summary",
    Data:     summaryData,
    Position: "A1",  // Excel-style position
    // ... other settings
})
```

### YAML File Example (`report_config.yaml`)

```yaml
version: "1.0"
name: "Employee Report"
sheets:
  - name: "Report"
    sections:
      - id: "employees"           # ID for BindSectionData()
        title: "Employees"
        locked: false
        direction: "horizontal"
        gap_after: 2
        title_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1565C0"
        header_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1976D2"
        columns:
          - field_name: "ID"
            header: "Employee ID"
            width: 12
          - field_name: "Name"
            header: "Full Name"
            width: 25

      - id: "managers"
        title: "Managers"
        locked: true
        direction: "horizontal"
```

### Go Usage

```go
// Load configuration from YAML file
exporter, err := pgexcel.NewDataExporterFromYamlFile("report_config.yaml")
if err != nil {
    log.Fatal(err)
}

// Bind data to sections by ID and export
err = exporter.
    BindSectionData("employees", employeeSlice).
    BindSectionData("managers", managerSlice).
    ExportToFile(ctx, "output.xlsx")
```

## Struct Tags

```go
type Product struct {
    ID    int     `excel:"header:Product ID,width:10"`
    Name  string  `excel:"header:Product Name,width:30"`
    Price float64 `excel:"header:Price ($),format:$#,##0.00"`
    SKU   string  `excel:"-"` // Hidden column
}
```

## Date/Time Formatting

Use Excel number format strings in `ColumnConfig.Format` to display `time.Time` values as dates:

### YAML Example
```yaml
columns:
  - field_name: "BirthDate"
    header: "Birth Date"
    format: "yyyy-mm-dd"       # -> 2024-12-06
  - field_name: "HireDate"
    header: "Hire Date"
    format: "dd/mm/yyyy"       # -> 06/12/2024
  - field_name: "CreatedAt"
    format: "yyyy-mm-dd hh:mm" # -> 2024-12-06 14:30
```

### Programmatic Example
```go
Columns: []pgexcel.ColumnConfig{
    {FieldName: "BirthDate", Format: "yyyy-mm-dd"},
    {FieldName: "HireDate", Format: "dd/mm/yyyy"},
}
```

### Common Excel Date Formats
| Format | Example Output |
|--------|----------------|
| `yyyy-mm-dd` | 2024-12-06 |
| `dd/mm/yyyy` | 06/12/2024 |
| `mm/dd/yyyy` | 12/06/2024 |
| `dd-mmm-yyyy` | 06-Dec-2024 |
| `yyyy-mm-dd hh:mm` | 2024-12-06 14:30 |


## Internal Flow

```
Export() 
  └── for each sheet in e.data:
        ├── if data is *sheetWithSections:
        │     └── exportSections()  ← Handles stacked sections
        └── else:
              └── exportSheet()     ← Handles single data slice

exportSections()
  └── for each section:
        ├── Determine direction (vertical/horizontal)
        ├── Calculate startCol, startRow
        ├── Write title (if any)
        ├── Write headers
        ├── Write data rows
        ├── Apply styles
        ├── Update position trackers
        └── Apply protection
```

## Breaking Change Prevention

### DO NOT Change:
1. **Public struct field names** in `SectionConfig`, `ColumnConfig`, `StyleTemplate`
2. **Method signatures** on `DataExporter` and `SheetBuilder`
3. **Default behaviors** (e.g., ShowHeader defaults to true)
4. **Direction values**: "vertical" and "horizontal" are string constants

### Safe to Change:
1. Internal helper functions (unexported)
2. Style defaults (colors, fonts)
3. Add new optional fields with zero-value defaults
4. Add new methods (doesn't break existing code)

## Testing

```bash
cd apigateway/pkg/pgexcel
go test -v
```

**Current tests:**
- `TestDataExporterWithStructs` - Basic struct export
- `TestDataExporterWithExcelTags` - Excel tag parsing
- `TestDataExporterWithMaps` - Map-based export
- `TestDataExporterMultipleSheets` - Multi-sheet export
- `TestDataExporterEmptySlice` - Empty data handling
- `TestDataExporterWithTemplate` - YAML template integration
- `TestParseExcelTag` - Tag parsing logic
- `TestExtractColumnsFromStruct` - Column extraction
- `TestEvaluateConditionDataExporter` - Conditional formatting
- `TestDataExporterWithStackedSections` - Vertical stacking
- `TestDataExporterWithSectionColumnOverrides` - Column overrides
- `TestDataExporterWithHorizontalSections` - Horizontal stacking
- `TestDataExporterWithYamlSections` - YAML section configuration with data binding

## Dependencies

- `github.com/xuri/excelize/v2` - Excel file manipulation
- `gopkg.in/yaml.v3` - YAML template parsing

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2024-12-06 | Initial consolidation from multiple files |
| 1.1 | 2024-12-06 | Added vertical section stacking |
| 1.2 | 2024-12-06 | Added horizontal section stacking |
| 1.3 | 2024-12-06 | Added YAML section configuration with `BindSectionData()` |
| 1.4 | 2024-12-06 | Added date/number format support via `ColumnConfig.Format` |
