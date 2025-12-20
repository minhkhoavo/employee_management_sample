# simpleexcel - Simple Data Exporter

A lightweight Go library for exporting data to Excel files with basic styling and layout support.

## Features

- **Simple API**: Easy-to-use fluent interface for building Excel exports
- **YAML Support**: Define templates with YAML for consistent report generation
- **Mixed Configuration**: Combine YAML templates with programmatic dynamic updates
- **Hidden Data**: Support for hidden columns (metadata) and hidden sections with distinct styling
- **Advanced Protection**: Smart cell locking (unused cells unlocked) and formatting permissions
- **Formatters**: Custom data formatting (e.g., currency, dates) via function registration
- **Flexible Layouts**: Position sections vertically or horizontally
- **Runtime Data Binding**: Bind data to templates at runtime

## Installation

```bash
go get github.com/your-org/your-repo/apigateway/pkg/simpleexcel
```

## Quick Start

### Programmatic Usage

```go
package main

import (
	"context"
	"github.com/your-org/your-repo/apigateway/pkg/simpleexcel"
)

type Employee struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func main() {
	// Sample data
	employees := []Employee{
		{1, "John Doe", "Developer"},
		{2, "Jane Smith", "Designer"},
	}

	// Create and configure exporter
	err := simpleexcel.NewDataExporter().
		AddSheet("Employees").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Team Members",
			Data:       employees,
			ShowHeader: true,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 15},
				{FieldName: "Name", Header: "Full Name", Width: 25},
				{FieldName: "Role", Header: "Position", Width: 20},
			},
		}).
		Build().
		ExportToExcel(context.Background(), "employees.xlsx")

	if err != nil {
		panic(err)
	}
}
```

### YAML Template Example

```yaml
# report.yaml
sheets:
  - name: "Employee Report"
    sections:
      - id: "employees"
        title: "Team Members"
        show_header: true
        direction: "vertical"
        title_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1565C0"
        columns:
          - field_name: "ID"
            header: "Employee ID"
            width: 15
          - field_name: "Name"
            header: "Full Name"
            width: 25
          - field_name: "Role"
            header: "Position"
            width: 20
```

### Using YAML Template

```go
import (
    "os"
    "github.com/your-org/your-repo/apigateway/pkg/simpleexcel"
)

// Read YAML file
data, err := os.ReadFile("report.yaml")
if err != nil {
    log.Fatal(err)
}

// Initialize exporter
tmpl, err := simpleexcel.NewDataExporterFromYamlConfig(string(data))
if err != nil {
    log.Fatal(err)
}

// Bind data to the section defined in YAML
tmpl.BindSectionData("employees", employees)

// Export to file
err = tmpl.ExportToExcel(context.Background(), "employee_report.xlsx")
if err != nil {
    log.Fatal(err)
}
```

## Advanced Features

### Hidden Data & Metadata
You can include data in your Excel report that is hidden from the user by default but can be unhidden for inspection or processing.

**Hidden Fields (Columns)**
Add a `HiddenFieldName` to any column configuration. This will generate a hidden row immediately below the section title containing these field names. This is useful for mapping Excel columns back to database fields.

**Hidden Sections**
Set a section's type to `hidden` (or `SectionTypeHidden` in Go).
- **Behavior**: Data rows in this section are automatically hidden.
- **Styling**: Hidden rows have a **yellow background** (`#FFFF00`) by default to distinguish them as metadata when unhidden.

### Sheet Protection
When `Locked: true` is set on any section or column:
1.  **Unused Cells Unlocked**: All cells outside the specific report sections are automatically **unlocked**. Users can freely add data to the rest of the sheet.
2.  **Report Integrity**: Cells within `Locked` sections are read-only.
3.  **Hidden Row Locking**: Hidden metadata rows are explicitly locked to prevent tampering, even if unhidden.
4.  **Formatting Allowed**: Row and Column formatting is enabled in protected sheets, allowing users to **hide/unhide** rows to view metadata.

### Mixed Configuration (YAML + Fluent)
You can load a base template from YAML and then extend it programmatically.

```go
// 1. Load from YAML
exporter, _ := simpleexcel.NewDataExporterFromYamlConfig(yamlConfig)

// 2. Bind Data to YAML sections
exporter.BindSectionData("employees", employees)

// 3. Extend programmatically
if sheet := exporter.GetSheet("Employee Report"); sheet != nil {
    sheet.AddSection(&simpleexcel.SectionConfig{
        Title: "Debug Info",
        Type:  simpleexcel.SectionTypeHidden,
        Data:  debugData,
        // ...
    })
}
```

## API Reference

### DataExporter

#### Constructors
- `NewDataExporter()` - Creates a new DataExporter instance
- `NewDataExporterFromYamlConfig(config string)` - Creates a DataExporter from a YAML string

#### Methods
- `AddSheet(name string) *SheetBuilder` - Start building a new sheet
- `GetSheet(name string) *SheetBuilder` - Retrieve an existing sheet by name
- `GetSheetByIndex(index int) *SheetBuilder` - Retrieve an existing sheet by index
- `RegisterFormatter(name string, fn func(interface{}) interface{})` - Register a value formatter
- `BindSectionData(id string, data interface{}) *DataExporter` - Bind data to a YAML section
- `ExportToExcel(ctx context.Context, path string) error` - Export to Excel file
- `ToBytes() ([]byte, error)` - Export to in-memory byte slice

### SectionConfig

### Configuration Structs

```go
type SectionConfig struct {
    ID          string         `yaml:"id"`
    Title       string         `yaml:"title"`
    Type        string         `yaml:"type"`      // "full" (default) or "hidden"
    Data        interface{}    `yaml:"-"`
    Locked      bool           `yaml:"locked"`
    ShowHeader  bool           `yaml:"show_header"`
    Direction   string         `yaml:"direction"` // "horizontal" or "vertical"
    Position    string         `yaml:"position"`  // e.g., "A1"
    TitleStyle  *StyleTemplate `yaml:"title_style"`
    HeaderStyle *StyleTemplate `yaml:"header_style"`
    DataStyle   *StyleTemplate `yaml:"data_style"`
    Columns     []ColumnConfig `yaml:"columns"`
}

type ColumnConfig struct {
    FieldName       string  `yaml:"field_name"`
    HiddenFieldName string  `yaml:"hidden_field_name"` // Metadata field name (hidden row)
    Header          string  `yaml:"header"`
    Width           float64 `yaml:"width"`
    Locked          *bool   `yaml:"locked"`
    Formatter       string  `yaml:"formatter"` // Name of registered formatter
}
```

## Performance Considerations

1. **Memory Usage**: The Excel file is built in memory. For very large exports, consider:
   - Using pagination
   - Implementing a streaming approach with `excelize.StreamWriter`
   - Using server-side processing with progress updates

2. **Timeouts**: Set appropriate timeouts for large exports:
   ```go
   // In your route handler
   ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
   defer cancel()
   
   // Pass this context to your data fetching logic
   data, err := fetchLargeDataset(ctx)
   ```

3. **Caching**: For frequently generated reports, consider caching the generated file.


## Best Practices

1. **Error Handling**: Always handle errors from exporter methods
2. **Memory Management**: For large exports, consider streaming to disk first
3. **Content Type Headers**: Always set appropriate content type headers
4. **File Names**: Use meaningful filenames with proper extensions
5. **Timeouts**: Consider adding timeouts for large exports

## Performance Tips

1. **Reuse Exporter**: For repeated exports with the same structure
2. **Batch Processing**: For large datasets, process in batches
3. **Background Jobs**: For very large exports, consider using a job queue
4. **Caching**: Cache generated reports when possible

## License

[MIT](LICENSE)
