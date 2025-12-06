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

func TestDataExporterWithStackedSections(t *testing.T) {
	// Section 1: Employees (locked)
	type Employee struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// Section 2: Notes (editable, different structure)
	type Note struct {
		Text     string `json:"text"`
		Priority string `json:"priority"`
	}

	employees := []Employee{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	notes := []Note{
		{Text: "Task A", Priority: "High"},
		{Text: "Task B", Priority: "Low"},
	}

	exporter := NewDataExporter().
		AddSheet("Report").
		AddSection(&SectionConfig{
			Title:  "Employees (Read-Only)",
			Data:   employees,
			Locked: true,
			TitleStyle: &StyleTemplate{
				Font: &FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &FillTemplate{Color: "#2E7D32"},
			},
			HeaderStyle: &StyleTemplate{
				Font: &FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &FillTemplate{Color: "#4CAF50"},
			},
			GapAfter: 2,
		}).
		AddSection(&SectionConfig{
			Title:  "Notes (Editable)",
			Data:   notes,
			Locked: false,
			TitleStyle: &StyleTemplate{
				Font: &FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &FillTemplate{Color: "#1565C0"},
			},
			HeaderStyle: &StyleTemplate{
				Font: &FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &FillTemplate{Color: "#1976D2"},
			},
		}).
		Build()

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export with sections failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterWithSectionColumnOverrides(t *testing.T) {
	type Item struct {
		Code  string
		Name  string
		Value float64
	}

	items := []Item{
		{Code: "A001", Name: "Item A", Value: 100.50},
		{Code: "B002", Name: "Item B", Value: 200.75},
	}

	exporter := NewDataExporter().
		AddSheet("Items").
		AddSection(&SectionConfig{
			Title: "Inventory",
			Data:  items,
			Columns: []ColumnConfig{
				{FieldName: "Code", Header: "Item Code", Width: 15},
				{FieldName: "Name", Header: "Description", Width: 30},
				{FieldName: "Value", Header: "Unit Price ($)", Width: 20},
			},
		}).
		Build()

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export with column overrides failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterWithHorizontalSections(t *testing.T) {
	type Revenue struct {
		Month  string
		Amount float64
	}

	type Expenses struct {
		Category string
		Value    float64
	}

	revenue := []Revenue{
		{Month: "Jan", Amount: 10000},
		{Month: "Feb", Amount: 12000},
	}

	expenses := []Expenses{
		{Category: "Rent", Value: 2000},
		{Category: "Salaries", Value: 5000},
	}

	exporter := NewDataExporter().
		AddSheet("Dashboard").
		AddSection(&SectionConfig{
			Title:     "Revenue",
			Data:      revenue,
			Direction: "horizontal",
			GapAfter:  1,
		}).
		AddSection(&SectionConfig{
			Title:     "Expenses",
			Data:      expenses,
			Direction: "horizontal",
		}).
		Build()

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export with horizontal sections failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestDataExporterWithYamlSections(t *testing.T) {
	type Employee struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}

	type Manager struct {
		ID   int    `json:"id"`
		Role string `json:"role"`
	}

	employees := []Employee{
		{ID: 1, Name: "Alice", Status: "Active"},
		{ID: 2, Name: "Bob", Status: "Inactive"},
	}

	managers := []Manager{
		{ID: 1, Role: "Team Lead"},
		{ID: 2, Role: "Director"},
	}

	yamlConfig := `
version: "1.0"
name: "Test Report"
sheets:
  - name: "Dashboard"
    sections:
      - id: "emp_section"
        title: "Employees"
        locked: false
        direction: "horizontal"
        gap_after: 1
        header_style:
          font:
            bold: true
            color: "#FFFFFF"
          fill:
            color: "#1976D2"
        columns:
          - field_name: "ID"
            header: "Employee ID"
            width: 15
          - field_name: "Name"
            header: "Full Name"
            width: 25
          - field_name: "Status"
            header: "Status"
            width: 15

      - id: "mgr_section"
        title: "Managers"
        locked: true
        direction: "horizontal"
        header_style:
          font:
            bold: true
          fill:
            color: "#4CAF50"
        columns:
          - field_name: "ID"
            header: "Manager ID"
            width: 15
          - field_name: "Role"
            header: "Role Title"
            width: 25
`

	exporter, err := NewDataExporterFromYaml(yamlConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter from YAML: %v", err)
	}

	exporter.
		BindSectionData("emp_section", employees).
		BindSectionData("mgr_section", managers)

	var buf bytes.Buffer
	err = exporter.Export(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Export with YAML sections failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}
