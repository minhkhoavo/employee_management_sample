package simpleexcelv2

import (
	"testing"
)

func TestDataExporter_HiddenRowLocked(t *testing.T) {
	exporter := NewExcelDataExporter()

	type Product struct {
		Name string
	}
	data := []Product{{"Product A"}}

	exporter.AddSheet("HiddenLockTest").
		AddSection(&SectionConfig{
			Title:      "Locked Hidden Row",
			ShowHeader: true,
			Data:       data,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Name", HiddenFieldName: "db_name"},
			},
		})

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	sheetName := "HiddenLockTest"

	// Hidden Row should be Row 2
	val, _ := excelFile.GetCellValue(sheetName, "A2")
	if val != "db_name" {
		t.Errorf("Expected hidden field value 'db_name', got '%s'", val)
	}

	// Verify Locked Status
	styleID, err := excelFile.GetCellStyle(sheetName, "A2")
	if err != nil {
		t.Errorf("Failed to get cell style: %v", err)
	}

	style, err := excelFile.GetStyle(styleID)
	if err != nil {
		t.Errorf("Failed to get style details: %v", err)
	}

	if style.Protection == nil || !style.Protection.Locked {
		t.Errorf("Expected hidden cell to be locked")
	}
}
