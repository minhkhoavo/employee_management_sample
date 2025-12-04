package pgexcel

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

func TestDataExporterWithStructs(t *testing.T) {
	type Employee struct {
		ID     int     `json:"id"`
		Name   string  `json:"name"`
		Salary float64 `json:"salary"`
	}

	employees := []Employee{
		{ID: 1, Name: "Alice", Salary: 50000},
		{ID: 2, Name: "Bob", Salary: 60000},
	}

	exporter := NewDataExporter().
		WithData("Employees", employees)

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterWithExcelTags(t *testing.T) {
	type Product struct {
		ID    int     `excel:"header:Product ID,width:10"`
		Name  string  `excel:"header:Product Name,width:30"`
		Price float64 `excel:"header:Price ($),format:$#,##0.00"`
		SKU   string  `excel:"-"` // Hidden
	}

	products := []Product{
		{ID: 1, Name: "Widget", Price: 29.99, SKU: "WID001"},
		{ID: 2, Name: "Gadget", Price: 49.99, SKU: "GAD001"},
	}

	exporter := NewDataExporter().
		WithData("Products", products)

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterWithMaps(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 30},
		{"id": 2, "name": "Bob", "age": 25},
	}

	exporter := NewDataExporter().
		WithData("Users", data)

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterMultipleSheets(t *testing.T) {
	type Employee struct {
		Name string
		Dept string
	}

	type Department struct {
		Name  string
		Count int
	}

	employees := []Employee{
		{Name: "Alice", Dept: "Engineering"},
		{Name: "Bob", Dept: "Sales"},
	}

	departments := []Department{
		{Name: "Engineering", Count: 10},
		{Name: "Sales", Count: 5},
	}

	exporter := NewDataExporter().
		WithData("Employees", employees).
		WithData("Departments", departments)

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterEmptySlice(t *testing.T) {
	type Empty struct {
		ID int
	}

	exporter := NewDataExporter().
		WithData("Empty", []Empty{})

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
}

func TestDataExporterWithTemplate(t *testing.T) {
	type Employee struct {
		ID     int
		Name   string
		Status string
	}

	employees := []Employee{
		{ID: 1, Name: "Alice", Status: "ACTIVE"},
		{ID: 2, Name: "Bob", Status: "INACTIVE"},
	}

	yamlTemplate := `
version: "1.0"
name: "Employee Report"
sheets:
  - name: "Employees"
    query: "unused"  # Required by validator but ignored for data export
    columns:
      - name: "ID"
        header: "Employee ID"
        width: 10
      - name: "Name"
        header: "Full Name"
        width: 25
      - name: "Status"
        conditional:
          - condition: "== 'ACTIVE'"
            style:
              font:
                color: "#008000"
    layout:
      freeze_rows: 1
      auto_filter: true
`

	template, err := LoadTemplateFromString(yamlTemplate)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	exporter := NewDataExporterWithTemplate(template).
		WithData("Employees", employees)

	var buf bytes.Buffer
	err = exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestParseExcelTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected ColumnInfo
	}{
		{
			tag:      "header:Name,width:20",
			expected: ColumnInfo{Header: "Name", Width: 20},
		},
		{
			tag:      "header:Price,format:$#,##0.00",
			expected: ColumnInfo{Header: "Price", Format: "$#,##0.00"},
		},
		{
			tag:      "-",
			expected: ColumnInfo{Hidden: true},
		},
	}

	e := &DataExporter{}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			col := ColumnInfo{}
			e.parseExcelTag(&col, tt.tag)

			if tt.expected.Header != "" && col.Header != tt.expected.Header {
				t.Errorf("Expected header %s, got %s", tt.expected.Header, col.Header)
			}
			if tt.expected.Width != 0 && col.Width != tt.expected.Width {
				t.Errorf("Expected width %f, got %f", tt.expected.Width, col.Width)
			}
			if tt.expected.Hidden != col.Hidden {
				t.Errorf("Expected hidden %v, got %v", tt.expected.Hidden, col.Hidden)
			}
		})
	}
}

func TestExtractColumnsFromStruct(t *testing.T) {
	type TestStruct struct {
		PublicField  string
		privateField string
		TaggedField  string `excel:"header:Custom Header"`
		JSONField    string `json:"json_name"`
	}

	val := TestStruct{
		PublicField:  "pub",
		privateField: "priv",
		TaggedField:  "tagged",
		JSONField:    "json",
	}

	e := &DataExporter{}
	columns := e.extractColumnsFromStruct(reflect.ValueOf(val), nil)

	// Should have 3 exported fields (privateField is unexported)
	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	// Check custom header from excel tag
	for _, col := range columns {
		if col.FieldName == "TaggedField" && col.Header != "Custom Header" {
			t.Errorf("Expected custom header 'Custom Header', got '%s'", col.Header)
		}
		if col.FieldName == "JSONField" && col.Header != "json_name" {
			t.Errorf("Expected JSON header 'json_name', got '%s'", col.Header)
		}
	}
}

func TestEvaluateConditionDataExporter(t *testing.T) {
	tests := []struct {
		value     interface{}
		condition string
		expected  bool
	}{
		{100, "> 50", true},
		{"ACTIVE", "== 'ACTIVE'", true},
		{"hello world", "contains 'world'", true},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			result := evaluateCondition(tt.value, tt.condition)
			if result != tt.expected {
				t.Errorf("evaluateCondition(%v, %s) = %v, expected %v",
					tt.value, tt.condition, result, tt.expected)
			}
		})
	}
}
