package simpleexcel

import (
	"testing"
)

func TestDataExporter_YamlHiddenFieldStyle(t *testing.T) {
	// Temporarily create a YAML file for testing or use existing one if suitable
	// Using existing report_config.yaml which we just modified is risky if paths differ in test environment
	// Safest is to create a temp one.

	yamlConfig := `
sheets:
  - name: "HiddenStyleTest"
    sections:
      - id: "sec1"
        title: "Test Section"
        type: "full"
        show_header: true
        columns:
          - field_name: "Name"
            header: "Name"
            hidden_field_name: "db_name"
`
	exporter, err := NewDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter from yaml: %v", err)
	}

	data := []struct{ Name string }{{"Test Item"}}
	exporter.BindSectionData("sec1", data)

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	sheetName := "HiddenStyleTest"

	// Verify Hidden Row Value
	val, _ := excelFile.GetCellValue(sheetName, "A2")
	if val != "db_name" {
		t.Errorf("Expected hidden field value 'db_name', got '%s'", val)
	}

	// Verify Style (Simple check: GetCellStyle should return an ID > 0)
	styleID, err := excelFile.GetCellStyle(sheetName, "A2")
	if err != nil {
		t.Errorf("Failed to get cell style: %v", err)
	}
	if styleID == 0 {
		t.Errorf("Expected style ID > 0 for hidden cell, got 0")
	}

	// Verify Visibility
	visible, _ := excelFile.GetRowVisible(sheetName, 2)
	if visible {
		t.Errorf("Row 2 should be hidden")
	}
}
