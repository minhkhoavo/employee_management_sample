package simpleexcel

import (
	"testing"
)

func TestDataExporter_HiddenSectionStyle(t *testing.T) {
	exporter := NewDataExporter()

	type Product struct {
		Name  string
		Price float64
	}

	data := []Product{
		{"Hidden Item", 10.0},
	}

	exporter.AddSheet("HiddenSectionTest").
		AddSection(&SectionConfig{
			Title:      "Hidden Section",
			Type:       SectionTypeHidden, // This should trigger the default style
			ShowHeader: true,
			Data:       data,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Name"},
				{FieldName: "Price", Header: "Price"},
			},
		})

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	sheetName := "HiddenSectionTest"

	// Logic:
	// Row 1: Title
	// Row 2: Header
	// Row 3: Data (Should be hidden and styled)

	// Check Data Row (Row 3)
	val, _ := excelFile.GetCellValue(sheetName, "A3")
	if val != "Hidden Item" {
		t.Errorf("Expected data value 'Hidden Item', got '%s'", val)
	}

	// Verify Style (Non-zero ID implies style applied)
	styleID, err := excelFile.GetCellStyle(sheetName, "A3")
	if err != nil {
		t.Errorf("Failed to get cell style: %v", err)
	}
	if styleID == 0 {
		t.Errorf("Expected style ID > 0 for hidden data row, got 0")
	}

	// Verify Visibility (Should be hidden because of SectionTypeHidden)
	visible, _ := excelFile.GetRowVisible(sheetName, 3)
	if visible {
		t.Errorf("Row 3 should be hidden")
	}
}
