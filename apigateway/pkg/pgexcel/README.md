# PgExcel - PostgreSQL to Excel Export Library

A production-grade Golang library for exporting PostgreSQL query results to Excel files with advanced cell protection capabilities.

## Features

- 🔐 **Advanced Protection** - Lock/unlock specific cells, rows, columns, or ranges
- 🗄️ **Database Integration** - Execute any PostgreSQL query and export results
- 📊 **Multi-Sheet Support** - Create workbooks with multiple sheets from different queries
- 🎨 **Rich Styling** - Customizable headers, data formatting, and cell styles
- ⚡ **High Performance** - Efficient streaming for large datasets
- 🛠️ **Flexible API** - Builder pattern and functional options for maximum flexibility

## Installation

```bash
go get employee_management_sample/pkg/pgexcel
go get github.com/xuri/excelize/v2
go get github.com/lib/pq
```

## Quick Start

### Basic Export

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    _ "github.com/lib/pq"
    "employee_management_sample/pkg/pgexcel"
)

func main() {
    db, _ := sql.Open("postgres", "postgres://user:pass@localhost/db")
    defer db.Close()
    
    query := "SELECT id, name, email FROM employees"
    
    exporter := pgexcel.NewExporter(db).
        WithQuery(query).
        WithSheetName("Employees").
        WithPassword("readonly")
    
    exporter.ExportToFile(context.Background(), "employees.xlsx",
        pgexcel.WithAutoFilter(),
        pgexcel.WithFreezePanes(),
```

### Export Go Data (No Database Required)

```go
type Employee struct {
    ID     int     `excel:"header:Employee ID,width:10"`
    Name   string  `excel:"header:Full Name,width:25"`
    Salary float64 `excel:"header:Salary,format:$#,##0.00"`
    Hidden string  `excel:"-"` // Skip this field
}

employees := []Employee{
    {ID: 1, Name: "Alice", Salary: 75000},
    {ID: 2, Name: "Bob", Salary: 85000},
}

exporter := pgexcel.NewDataExporter().
    WithData("Employees", employees)

exporter.ExportToFile(ctx, "employees.xlsx")
```

### Export with YAML Template (Data Mode)

```go
// Load template and use with in-memory data
exporter, _ := pgexcel.NewDataExporterFromTemplateFile("template.yaml")
exporter.WithData("Employees", employees)
exporter.ExportToFile(ctx, "report.xlsx")
```

### Export Multiple Data Sources

```go
exporter := pgexcel.NewDataExporter().
    WithData("Employees", employees).
    WithData("Departments", departments).
    WithData("Summary", summaryData)

exporter.ExportToFile(ctx, "report.xlsx")
```

### Export with Editable Columns

```go
// Lock all cells except Status (E) and Comments (F) columns
exporter := pgexcel.NewExporter(db).
    WithQuery(query).
    WithProtection(
        pgexcel.LockAllExcept(
            pgexcel.Columns("E", "F"),
        ),
    ).
    WithPassword("edit123")

exporter.ExportToFile(ctx, "report.xlsx",
    pgexcel.WithAutoFilter(),
    pgexcel.WithAutoFitColumns(),
)
```

## Protection Patterns

### Lock All Except Specific Columns

```go
pgexcel.LockAllExcept(
    pgexcel.Columns("D", "E", "F"), // Columns D, E, F are editable
)
```

### Lock Specific Ranges

```go
pgexcel.LockRanges("A1:C100", "G1:G100")
```

### Unlock Specific Ranges

```go
pgexcel.UnlockRange("D2:E1000") // Allow editing in this range
```

### Lock Specific Rows

```go
pgexcel.LockRows(1, 2, 3)           // Lock specific rows
pgexcel.LockRowsAbove(5)            // Lock rows 1-5
pgexcel.LockRowsBelow(100)          // Lock rows 100 and below
```

### Lock Specific Columns

```go
pgexcel.LockColumns("A", "B", "C")
```

### Conditional Row Locking

```go
// Lock rows where role is "ADMIN"
pgexcel.LockRowsWhere(func(rowNum int, rowData []interface{}) bool {
    role := rowData[2].(string)
    return role == "ADMIN"
})
```

### Conditional Cell Locking

```go
// Lock cells based on value
pgexcel.LockCellsWhere(func(col string, rowNum int, value interface{}) bool {
    return value.(float64) > 100000 // Lock high values
})
```

### Combine Multiple Rules

```go
protection := pgexcel.CombineRules(
    pgexcel.LockRowsAbove(1),              // Lock header
    pgexcel.LockColumns("A", "B"),         // Lock ID columns
    pgexcel.UnlockRange("C2:E1000"),       // Allow editing data
)
```

## Styling

### Pre-defined Header Styles

```go
pgexcel.WithHeaderStyle(pgexcel.HeaderStyleBlue())
pgexcel.WithHeaderStyle(pgexcel.HeaderStyleGreen())
pgexcel.WithHeaderStyle(pgexcel.HeaderStyleDark())
```

### Custom Styles with Builder

```go
customHeader := pgexcel.NewStyleBuilder().
    Font("Calibri", 12).
    Bold().
    FontColor("#FFFFFF").
    Fill("#FF6B6B").
    Align("center").
    Build()

exporter.ExportToFile(ctx, "report.xlsx",
    pgexcel.WithHeaderStyle(customHeader),
)
```

### Column-specific Styles

```go
pgexcel.WithColumnStyle("salary", pgexcel.CurrencyStyle("$"))
pgexcel.WithColumnStyle("percentage", pgexcel.PercentageStyle())
pgexcel.WithColumnStyle("hire_date", pgexcel.DateStyle("2006-01-02"))
```

### Editable vs Read-only Styles

```go
pgexcel.WithColumnStyle("editable", pgexcel.DataStyleEditable())
pgexcel.WithColumnStyle("readonly", pgexcel.DataStyleReadOnly())
```

## Multi-Sheet Export

```go
exporter := pgexcel.NewExporter(db).
    WithQuery("SELECT * FROM summary").
    WithSheetName("Summary").
    WithPassword("summary123")

// Add more sheets
exporter.AddSheet(
    "SELECT * FROM details WHERE status = $1",
    "Details",
    pgexcel.WithQueryArgs("ACTIVE"),
    pgexcel.WithSheetProtectionRules("details123",
        pgexcel.LockAllExcept(pgexcel.Columns("E")),
    ),
)

exporter.AddSheet(
    "SELECT * FROM actions",
    "Actions",
    // No protection - fully editable
)

exporter.ExportToFile(ctx, "report.xlsx")
```

## Advanced Options

### Export Options

```go
exporter.ExportToFile(ctx, "report.xlsx",
    pgexcel.WithAutoFilter(),           // Enable auto-filter
    pgexcel.WithFreezePanes(),          // Freeze header row
    pgexcel.WithAutoFitColumns(),       // Auto-fit column widths
    pgexcel.WithMaxColumnWidth(50),     // Set max column width
    pgexcel.WithHeaders(true),          // Include headers (default: true)
    pgexcel.WithDateFormat("2006-01-02"),
    pgexcel.WithNumberFormat("#,##0.00"),
)
```

### Protection Options

```go
protection := pgexcel.NewSheetProtection()
protection.Password = "secure123"
protection.AllowFilter = true
protection.AllowSort = true
protection.AllowFormatCells = false
protection.AllowInsertRows = false
protection.AllowDeleteRows = false

exporter.config.Protection = protection
```

## Real-World Examples

### Payroll Report - Complex Protection

```go
// Allow editing base salary and bonus, lock everything else
protection := pgexcel.CombineRules(
    pgexcel.LockRowsAbove(1),                    // Lock header
    pgexcel.LockColumns("A", "B", "C", "F", "G"), // Lock ID, name, total
    pgexcel.UnlockRange("D2:E1000"),             // Allow salary/bonus edit
)

exporter := pgexcel.NewExporter(db).
    WithQuery("SELECT * FROM payroll").
    WithSheetName("Payroll").
    WithProtection(protection).
    WithPassword("payroll2024")

exporter.ExportToFile(ctx, "payroll.xlsx",
    pgexcel.WithColumnStyle("base_salary", pgexcel.CurrencyStyle("$")),
    pgexcel.WithColumnStyle("bonus", pgexcel.CurrencyStyle("$")),
)
```

### Financial Report - Multi-Layer Protection

```go
// Lock actuals, allow forecast editing
rules := pgexcel.LockAllExcept(
    pgexcel.Columns("F", "H"), // Forecast and Notes
)

exporter := pgexcel.NewExporter(db).
    WithQuery("SELECT * FROM financial_report").
    WithSheetName("Q4 Report").
    WithProtection(rules).
    WithPassword("finance2024")

exporter.ExportToFile(ctx, "financial.xlsx",
    pgexcel.WithHeaderStyle(pgexcel.HeaderStyleGreen()),
    pgexcel.WithColumnStyle("revenue", pgexcel.CurrencyStyle("$")),
    pgexcel.WithColumnStyle("variance", pgexcel.PercentageStyle()),
)
```

## API Reference

### Exporter Methods

- `NewExporter(db DB)` - Create new exporter
- `WithQuery(query string, args ...interface{})` - Set SQL query
- `WithSheetName(name string)` - Set sheet name
- `WithHeaders(bool)` - Enable/disable headers
- `WithProtection(...ProtectionRule)` - Set protection rules
- `WithPassword(password string)` - Set protection password
- `AddSheet(query, name, ...SheetOption)` - Add another sheet
- `Export(ctx, writer, ...ExportOption)` - Export to writer
- `ExportToFile(ctx, path, ...ExportOption)` - Export to file

### Protection Rules

- `LockAllExcept(...ProtectionRule)` - Lock all except specified
- `Columns(...string)` - Unlock columns
- `LockColumns(...string)` - Lock columns
- `UnlockRange(...string)` - Unlock ranges
- `LockRanges(...string)` - Lock ranges
- `LockRows(...int)` - Lock specific rows
- `LockRowsAbove(int)` - Lock rows above
- `LockRowsBelow(int)` - Lock rows below
- `LockRowsWhere(RowFilterFunc)` - Conditional row locking
- `LockCellsWhere(CellFilterFunc)` - Conditional cell locking
- `CombineRules(...ProtectionRule)` - Combine multiple rules

### Export Options

- `WithAutoFilter()` - Enable auto-filter
- `WithFreezePanes()` - Freeze header row
- `WithAutoFitColumns()` - Auto-fit column widths
- `WithMaxColumnWidth(int)` - Set max column width
- `WithHeaderStyle(*CellStyle)` - Set header style
- `WithColumnStyle(col, *CellStyle)` - Set column style
- `WithDateFormat(string)` - Set date format
- `WithTimeFormat(string)` - Set time format
- `WithNumberFormat(string)` - Set number format
- `WithProtection(*SheetProtection)` - Set protection config
- `WithProtectionRules(password, ...ProtectionRule)` - Build protection

### Style Helpers

- `NewStyleBuilder()` - Create style builder
- `HeaderStyleBlue()` - Blue header style
- `HeaderStyleGreen()` - Green header style
- `HeaderStyleDark()` - Dark header style
- `DataStyleEditable()` - Editable cell style
- `DataStyleReadOnly()` - Read-only cell style
- `CurrencyStyle(symbol)` - Currency formatting
- `PercentageStyle()` - Percentage formatting
- `DateStyle(format)` - Date formatting

## YAML Template System

The YAML Template System provides an XSLT-like capability for defining Excel report layouts dynamically via YAML configuration files.

### Quick Start with Templates

```go
// Load template from file
exporter, err := pgexcel.NewTemplateExporterFromFile(db, "templates/report.yaml")
if err != nil {
    log.Fatal(err)
}

// Set runtime variables
exporter.WithVariable("DEPARTMENT_ID", 1).
    WithVariable("REPORT_DATE", "2024-01-01")

// Export
err = exporter.ExportToFile(ctx, "report.xlsx")
```

### Template Structure

```yaml
version: "1.0"
name: "My Report"
description: "Report description"

defaults:
  header_style:
    font:
      name: "Arial"
      size: 11
      bold: true
      color: "#FFFFFF"
    fill:
      color: "#4472C4"
  data_style:
    font:
      name: "Arial"
      size: 10

variables:
  DEPARTMENT_ID: "1"

sheets:
  - name: "Employees"
    query: |
      SELECT id, name, salary, status 
      FROM employees WHERE department_id = $1
    query_args:
      - "${DEPARTMENT_ID}"
    
    columns:
      - name: "id"
        header: "ID"
        width: 8
      - name: "salary"
        format: "$#,##0.00"
        conditional:
          - condition: "> 100000"
            style:
              fill:
                color: "#FFE699"
      - name: "status"
        conditional:
          - condition: "== 'ACTIVE'"
            style:
              font:
                color: "#008000"
    
    layout:
      freeze_rows: 1
      auto_filter: true
    
    protection:
      password: "secret"
      lock_sheet: true
      unlocked_columns: ["status"]
      allow_filter: true
```

### Conditional Formatting

Support for simple expressions:
- **Numeric**: `> 100`, `< 50`, `>= 100`, `<= 100`, `== 100`, `!= 100`
- **String**: `== 'ACTIVE'`, `!= 'INACTIVE'`
- **Contains**: `contains 'error'`

### Template API

```go
// Load from file
exporter, _ := pgexcel.NewTemplateExporterFromFile(db, "report.yaml")

// Load from string
exporter, _ := pgexcel.NewTemplateExporterFromString(db, yamlContent)

// Load programmatically
template, _ := pgexcel.LoadTemplate("report.yaml")
exporter := pgexcel.NewTemplateExporter(db, template)

// Set variables
exporter.WithVariables(map[string]interface{}{
    "DEPT": 1,
    "DATE": "2024-01-01",
})

// Export
exporter.Export(ctx, writer)
exporter.ExportToFile(ctx, "output.xlsx")
```

## Best Practices

1. **Always use passwords** for protected sheets in production
2. **Use LockAllExcept** pattern for maximum security
3. **Apply auto-filter and freeze panes** for better UX
4. **Use pre-defined styles** for consistency
5. **Test protection** by trying to edit locked cells in Excel
6. **Use transactions** for consistency when exporting multiple related queries
7. **Handle context cancellation** for long-running exports
8. **Use YAML templates** for reports with complex/dynamic layouts

## License

This library is part of the employee_management_sample project.

