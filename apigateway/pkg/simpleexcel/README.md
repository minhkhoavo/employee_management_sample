# simpleexcel - Simple Data Exporter

A lightweight Go library for exporting data to Excel files with basic styling and layout support. This is a simplified version of the `pgexcel` package, focusing on core functionality.

## Features

- **Simple API**: Easy-to-use fluent interface for building Excel exports
- **YAML Support**: Define templates with YAML for consistent report generation
- **Basic Styling**: Support for fonts, colors, and cell locking
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
tmpl, err := simpleexcel.NewDataExporterFromYamlFile("report.yaml")
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

## API Reference

### DataExporter

#### Constructors
- `NewDataExporter()` - Creates a new DataExporter instance
- `NewDataExporterFromYamlFile(path string)` - Creates a DataExporter from a YAML template file

#### Methods
- `AddSheet(name string) *SheetBuilder` - Start building a new sheet
- `BindSectionData(id string, data interface{}) *DataExporter` - Bind data to a YAML section
- `ExportToExcel(ctx context.Context, path string) error` - Export to Excel file
- `StreamTo(w io.Writer) error` - Stream Excel file to a writer (memory efficient for large files)
- `StreamToResponse(w http.ResponseWriter, filename string) error` - Stream Excel file to HTTP response
- `ToCSV(w io.Writer) error` - Export first sheet as CSV (memory efficient)
- `ToCSVBytes() ([]byte, error)` - Export first sheet as CSV bytes

### SectionConfig

```go
type SectionConfig struct {
    ID          string         `yaml:"id"`
    Title       string         `yaml:"title"`
    Data        interface{}    `yaml:"-"`
    Locked      bool           `yaml:"locked"`
    ShowHeader  bool           `yaml:"show_header"`
    Direction   string         `yaml:"direction"` // "horizontal" or "vertical"
    Position    string         `yaml:"position"`  // e.g., "A1"
    TitleStyle  *StyleTemplate `yaml:"title_style"`
    Columns     []ColumnConfig `yaml:"columns"`
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

## Web Integration

### Streaming Large Exports

The package now includes built-in support for streaming large exports with minimal memory usage:

```go
// In your HTTP handler
exportHandler := func(w http.ResponseWriter, r *http.Request) {
    exporter := simpleexcel.NewDataExporter()
    // ... configure exporter ...
    
    // Stream directly to response
    if err := exporter.StreamToResponse(w, "large_export.xlsx"); err != nil {
        http.Error(w, "Export failed", http.StatusInternalServerError)
        return
    }
}
```

### Performance Considerations

- **Memory Efficiency**: The `StreamTo` and `StreamToResponse` methods use `excelize.StreamWriter` internally to handle large datasets with minimal memory usage.
- **Chunked Processing**: Data is processed in chunks (1000 rows at a time) to maintain low memory footprint.
- **Automatic Cleanup**: All resources are properly cleaned up, including temporary files.

For complete web framework examples (Echo, Gin, etc.), see the [WEB_INTEGRATION.md](WEB_INTEGRATION.md) file.

package main

import (
	"net/http"
	"github.com/labstack/echo/v4"
	"your-module-path/pkg/simpleexcel"
)

func main() {
	e := echo.New()

	// Excel Export
	e.GET("/export/excel", func(c echo.Context) error {
		// Sample data
		data := []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		}{
			{1, "John Doe", "Developer"},
			{2, "Jane Smith", "Designer"},
		}

		exporter := simpleexcel.NewDataExporter().
			AddSheet("Employees").
			AddSection(&simpleexcel.SectionConfig{
				Title:      "Team Members",
				Data:       data,
				ShowHeader: true,
				Columns: []simpleexcel.ColumnConfig{
					{FieldName: "ID", Header: "Employee ID", Width: 15},
					{FieldName: "Name", Header: "Full Name", Width: 25},
					{FieldName: "Role", Header: "Position", Width: 20},
				},
			}).
			Build()

		// Set headers for file download
		c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="employees.xlsx"`)
		
		// Stream directly to response
		return exporter.ToWriter(c.Response().Writer)
	})

	// CSV Export
	e.GET("/export/csv", func(c echo.Context) error {
		exporter := simpleexcel.NewDataExporter()
		// ... configure exporter ...

		c.Response().Header().Set(echo.HeaderContentType, "text/csv")
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="report.csv"`)
		
		return exporter.ToCSV(c.Response().Writer)
	})

	e.Logger.Fatal(e.Start(":1323"))
}
```

### Gin Framework

```go
package main

import (
	"github.com/gin-gonic/gin"
	"your-module-path/pkg/simpleexcel"
)

func main() {
	r := gin.Default()

	// Excel Export
	r.GET("/export/excel", func(c *gin.Context) {
		// Sample data
		data := []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		}{
			{1, "John Doe", "Developer"},
			{2, "Jane Smith", "Designer"},
		}

		exporter := simpleexcel.NewDataExporter().
			AddSheet("Employees").
			AddSection(&simpleexcel.SectionConfig{
				Title:      "Team Members",
				Data:       data,
				ShowHeader: true,
				Columns: []simpleexcel.ColumnConfig{
					{FieldName: "ID", Header: "Employee ID", Width: 15},
					{FieldName: "Name", Header: "Full Name", Width: 25},
					{FieldName: "Role", Header: "Position", Width: 20},
				},
			}).
			Build()

		// Stream directly to response
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", `attachment; filename="employees.xlsx"`)
		
		exporter.ToWriter(c.Writer)
	})

	// CSV Export
	r.GET("/export/csv", func(c *gin.Context) {
		exporter := simpleexcel.NewDataExporter()
		// ... configure exporter ...

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", `attachment; filename="report.csv"`)
		
		exporter.ToCSV(c.Writer)
	})

	r.Run(":8080")
}
```

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
