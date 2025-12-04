// Package main demonstrates exporting Go structs/slices to Excel
// without requiring a database connection.
//
// Run: go run data_export.go_sample
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/pkg/pgexcel"
)

// Employee represents an employee with Excel formatting tags
type Employee struct {
	ID         int       `excel:"header:Employee ID,width:10"`
	Name       string    `excel:"header:Full Name,width:25"`
	Email      string    `excel:"header:Email Address,width:30"`
	Department string    `excel:"header:Department,width:20"`
	Salary     float64   `excel:"header:Salary,format:$#,##0.00,width:15"`
	HireDate   time.Time `excel:"header:Hire Date,format:2006-01-02,width:12"`
	Status     string    `excel:"header:Status,width:10"`
	SSN        string    `excel:"-"` // Hidden - will not be exported
}

// Department represents a department summary
type Department struct {
	Name          string  `json:"name"`           // Uses json tag as fallback
	EmployeeCount int     `json:"employee_count"`
	TotalSalary   float64 `excel:"header:Total Salary,format:$#,##0.00"`
	AvgSalary     float64 `excel:"header:Avg Salary,format:$#,##0.00"`
}

func main() {
	ctx := context.Background()

	// Example 1: Basic struct export
	basicExport(ctx)

	// Example 2: Multiple sheets with different data types
	multiSheetExport(ctx)

	// Example 3: Export maps
	mapExport(ctx)

	// Example 4: Export with YAML template styling
	templateStyledExport(ctx)
}

// basicExport demonstrates the simplest way to export structs
func basicExport(ctx context.Context) {
	fmt.Println("Example 1: Basic struct export")

	employees := []Employee{
		{
			ID:         1,
			Name:       "Alice Johnson",
			Email:      "alice@company.com",
			Department: "Engineering",
			Salary:     85000,
			HireDate:   time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			Status:     "ACTIVE",
			SSN:        "123-45-6789", // Won't be exported
		},
		{
			ID:         2,
			Name:       "Bob Smith",
			Email:      "bob@company.com",
			Department: "Sales",
			Salary:     72000,
			HireDate:   time.Date(2019, 7, 1, 0, 0, 0, 0, time.UTC),
			Status:     "ACTIVE",
			SSN:        "987-65-4321",
		},
		{
			ID:         3,
			Name:       "Carol Williams",
			Email:      "carol@company.com",
			Department: "Marketing",
			Salary:     68000,
			HireDate:   time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC),
			Status:     "INACTIVE",
			SSN:        "555-55-5555",
		},
	}

	exporter := pgexcel.NewDataExporter().
		WithData("Employees", employees)

	if err := exporter.ExportToFile(ctx, "basic_export_output.xlsx"); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	fmt.Println("  -> Created basic_export_output.xlsx")
}

// multiSheetExport demonstrates exporting multiple data types to different sheets
func multiSheetExport(ctx context.Context) {
	fmt.Println("Example 2: Multi-sheet export")

	employees := []Employee{
		{ID: 1, Name: "Alice", Department: "Engineering", Salary: 85000, Status: "ACTIVE"},
		{ID: 2, Name: "Bob", Department: "Engineering", Salary: 92000, Status: "ACTIVE"},
		{ID: 3, Name: "Carol", Department: "Sales", Salary: 72000, Status: "ACTIVE"},
	}

	departments := []Department{
		{Name: "Engineering", EmployeeCount: 2, TotalSalary: 177000, AvgSalary: 88500},
		{Name: "Sales", EmployeeCount: 1, TotalSalary: 72000, AvgSalary: 72000},
	}

	exporter := pgexcel.NewDataExporter().
		WithData("Employees", employees).
		WithData("Summary", departments)

	if err := exporter.ExportToFile(ctx, "multi_sheet_output.xlsx"); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	fmt.Println("  -> Created multi_sheet_output.xlsx")
}

// mapExport demonstrates exporting map data
func mapExport(ctx context.Context) {
	fmt.Println("Example 3: Map data export")

	// Export slice of maps
	data := []map[string]interface{}{
		{"id": 1, "product": "Widget A", "price": 29.99, "stock": 150},
		{"id": 2, "product": "Widget B", "price": 49.99, "stock": 75},
		{"id": 3, "product": "Gadget X", "price": 99.99, "stock": 30},
	}

	exporter := pgexcel.NewDataExporter().
		WithData("Products", data)

	if err := exporter.ExportToFile(ctx, "map_export_output.xlsx"); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	fmt.Println("  -> Created map_export_output.xlsx")
}

// templateStyledExport demonstrates using YAML template with in-memory data
func templateStyledExport(ctx context.Context) {
	fmt.Println("Example 4: Template-styled export")

	// YAML template for styling (can also load from file)
	yamlTemplate := `
version: "1.0"
name: "Styled Employee Report"
sheets:
  - name: "Employees"
    query: "unused"  # Required but ignored for data export
    columns:
      - name: "ID"
        header: "🔢 ID"
        width: 8
        style:
          alignment: "center"
      - name: "Name"
        header: "👤 Employee Name"
        width: 25
        style:
          font:
            bold: true
      - name: "Salary"
        header: "💰 Salary"
        width: 15
        format: "$#,##0.00"
        conditional:
          - condition: "> 80000"
            style:
              fill:
                color: "#C6EFCE"
              font:
                color: "#006100"
          - condition: "< 50000"
            style:
              fill:
                color: "#FFC7CE"
              font:
                color: "#9C0006"
      - name: "Status"
        header: "📊 Status"
        width: 12
        conditional:
          - condition: "== 'ACTIVE'"
            style:
              font:
                color: "#008000"
                bold: true
          - condition: "== 'INACTIVE'"
            style:
              font:
                color: "#FF0000"
    layout:
      freeze_rows: 1
      auto_filter: true
    protection:
      password: "demo123"
      lock_sheet: true
      unlocked_columns:
        - "Status"
      allow_filter: true
`

	// Simple struct for this example
	type Employee struct {
		ID     int
		Name   string
		Salary float64
		Status string
	}

	employees := []Employee{
		{ID: 1, Name: "Alice Johnson", Salary: 85000, Status: "ACTIVE"},
		{ID: 2, Name: "Bob Smith", Salary: 45000, Status: "ACTIVE"},
		{ID: 3, Name: "Carol Williams", Salary: 92000, Status: "INACTIVE"},
		{ID: 4, Name: "David Brown", Salary: 68000, Status: "ACTIVE"},
	}

	template, err := pgexcel.LoadTemplateFromString(yamlTemplate)
	if err != nil {
		log.Fatalf("Failed to load template: %v", err)
	}

	exporter := pgexcel.NewDataExporterWithTemplate(template).
		WithData("Employees", employees)

	if err := exporter.ExportToFile(ctx, "styled_export_output.xlsx"); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	fmt.Println("  -> Created styled_export_output.xlsx")
	fmt.Println("     (Status column is editable, password: demo123)")
}
