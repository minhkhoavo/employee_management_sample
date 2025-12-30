# Web Framework Integration Guide

This guide shows how to use the `simpleexcel` package with popular Go web frameworks, with a focus on efficiently handling large exports using streaming.

## Table of Contents
- [Streaming Large Exports](#streaming-large-exports)
- [Echo Framework](#echo-framework)
- [Gin Framework](#gin-framework)
- [Performance Best Practices](#performance-best-practices)
- [Error Handling](#error-handling)

## Streaming Large Exports

The `simpleexcel` package provides efficient streaming capabilities for handling large exports with minimal memory usage. The key methods are:

- `ToWriter(w io.Writer) error` - Streams Excel data directly to any writer
- `ToCSV(w io.Writer) error` - Efficiently exports to CSV format

### Basic Streaming Example

```go
// Basic HTTP handler with streaming
exportHandler := func(w http.ResponseWriter, r *http.Request) {
    exporter := simpleexcel.NewDataExporter()
    
    // Configure your exporter
    exporter.AddSheet("Large Data").
        AddSection(&simpleexcel.SectionConfig{
            Title:      "Large Dataset",
            Data:       fetchLargeData(), // Your data fetching function
            ShowHeader: true,
            // ... other config
        })
    
    // Stream directly to response
    if err := exporter.ToWriter(w); err != nil {
        log.Printf("Export failed: %v", err)
        http.Error(w, "Export failed", http.StatusInternalServerError)
    }
}
```

## Echo Framework

### Excel Export

```go
// In your handler function
func exportEmployees(c echo.Context) error {
    // Your data
    data := []struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
        Role string `json:"role"`
    }{{
        ID: 1, Name: "John Doe", Role: "Developer",
    }}

    // Create and configure exporter
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
}
```

### CSV Export

```go
func exportCSV(c echo.Context) error {
    // ... configure exporter ...
    
    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="report.csv"`)
    
    return exporter.ToCSV(c.Response().Writer)
}
```

## Gin Framework

### Excel Export

```go
func exportExcel(c *gin.Context) {
    // ... configure exporter ...
    
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", `attachment; filename="employees.xlsx"`)
    
    exporter.ToWriter(c.Writer)
}
```

### CSV Export

```go
func exportCSV(c *gin.Context) {
    // ... configure exporter ...
    
    c.Header("Content-Type", "text/csv")
    c.Header("Content-Disposition", `attachment; filename="report.csv"`)
    
    exporter.ToCSV(c.Writer)
}
```

## Best Practices

1. **Error Handling**: Always handle errors from exporter methods
2. **Memory Management**: For large exports, consider streaming to disk first
3. **Content Type Headers**: Always set appropriate content type headers
4. **File Names**: Use meaningful filenames with proper extensions
5. **Timeouts**: Consider adding timeouts for large exports

### Bulk Data Export with `ToWriter`

For many scenarios, `ToWriter` combined with a pre-fetched dataset is sufficient and easy to implement.

#### Echo Framework

```go
func exportLargeData(c echo.Context) error {
    data := fetchEmployeesFromDB() // Returns []Employee
    
    exporter := simpleexcel.NewDataExporter().
        AddSheet("Employees").
        AddSection(&simpleexcel.SectionConfig{
            Title: "All Employees",
            Data:  data,
            ShowHeader: true,
        }).Build()

    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="employees.xlsx"`)
    
    return exporter.ToWriter(c.Response().Writer)
}
```

### Extreme Scale: CSV Export

When dealing with millions of rows where Excel's 1M row limit or memory overhead is an issue, use `ToCSV`.

```go
func streamLargeCSV(c echo.Context) error {
    data := fetchMillionsOfRows()
    
    exporter := simpleexcel.NewDataExporter().
        AddSheet("Report").
        AddSection(&simpleexcel.SectionConfig{
            Data: data,
        }).Build()

    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="large_report.csv"`)
    
    return exporter.ToCSV(c.Response().Writer)
}
```

### CSV Streaming for Very Large Datasets

For extremely large datasets, consider using CSV format which is more memory-efficient:

```go
func streamLargeCSV(c echo.Context) error {
    c.Response().Header().Set(echo.HeaderContentType, "text/csv")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="large_export.csv"`)
    
    // Create a writer that streams directly to the response
    w := csv.NewWriter(c.Response())
    
    // Write headers
    if err := w.Write([]string{"ID", "Name", "Email", "Department", "Salary"}); err != nil {
        return err
    }
    
    // Stream data in chunks
    for i := 1; i <= 1000000; i++ {
        // Get data in chunks (e.g., from database)
        row := []string{
            strconv.Itoa(i),
            fmt.Sprintf("Employee %d", i),
            fmt.Sprintf("employee%d@example.com", i),
            "Engineering",
            strconv.Itoa(rand.Intn(100000) + 50000),
        }
        
        if err := w.Write(row); err != nil {
            return err
        }
        
        // Flush periodically
        if i%1000 == 0 {
            w.Flush()
            if err := w.Error(); err != nil {
                return err
            }
        }
    }
    
    // Flush any remaining data
    w.Flush()
    return w.Error()
}
```

## Performance Best Practices

### Memory Management

1. **Use Streaming for Large Exports**
   - Always use `StreamTo` or `StreamToResponse` for large datasets
   - These methods process data in chunks (1000 rows at a time)

2. **CSV for Very Large Datasets**
   - For extremely large datasets, consider using CSV format
   - The `ToCSV` method is optimized for memory efficiency

### Web Server Configuration

1. **Timeouts**
   - Set appropriate timeouts for long-running exports
   - Example for Echo:
     ```go
     e.Server.WriteTimeout = 30 * time.Minute
     e.Server.ReadTimeout = 30 * time.Minute
     ```

2. **Response Compression**
   - Enable gzip compression for smaller network transfer
   - Example for Gin:
     ```go
     router.Use(gzip.Gzip(gzip.DefaultCompression))
     ```

### Background Processing

For very large exports, consider using a background job system:

```go
// Example using a simple background worker
go func() {
    // Process and save to storage
    file, _ := os.Create("/tmp/export.xlsx")
    defer file.Close()
    
    if err := exporter.StreamTo(file); err != nil {
        // Handle error
        return
    }
    
    // Notify user or update job status
    notifyUser(userID, "/downloads/export.xlsx")
}()
```

## Error Handling

### Common Errors

1. **Timeout Errors**
   - Handle context timeouts gracefully
   - Provide meaningful error messages to users

2. **Memory Issues**
   - Monitor memory usage
   - Implement circuit breakers for very large exports

3. **File System Errors**
   - Check disk space before starting large exports
   - Handle permission issues

### Example Error Handler

```go
func handleExportError(err error, c echo.Context) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return c.JSON(http.StatusRequestTimeout, map[string]string{
            "error": "Export took too long, please try with a smaller dataset",
        })
    }
    
    log.Printf("Export error: %v", err)
    return c.JSON(http.StatusInternalServerError, map[string]string{
        "error": "Failed to generate export",
    })
}
```

## Monitoring and Logging

1. **Log Export Events**
   - Log start/end of exports
   - Track export sizes and durations

2. **Metrics**
   - Track number of exports
   - Monitor memory usage
   - Track export durations

Example with Prometheus:
```go
var (
    exportDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "export_duration_seconds",
        Help:    "Time taken to generate exports",
        Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
    })
)

// In your handler
func exportHandler(c echo.Context) error {
    start := time.Now()
    defer func() {
        exportDuration.Observe(time.Since(start).Seconds())
    }()
    
    // ... export logic ...
}
```
