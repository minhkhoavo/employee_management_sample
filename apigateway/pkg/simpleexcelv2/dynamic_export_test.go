package simpleexcelv2

import (
	"context"
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestDynamicMapExport(t *testing.T) {
	// 1. Setup Dynamic Data (Slice of Maps)
	data := []map[string]interface{}{
		{"Name": "Product A", "Price": 100, "Features_Color": "Red"},
		{"Name": "Product B", "Price": 200, "Features_Color": "Blue"},
	}

	// 2. Create Exporter
	exporter := NewExcelDataExporter().
		AddSheet("DynamicSheet").
		AddSection(&SectionConfig{
			Title:      "Dynamic Data",
			Data:       data,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 20},
				{FieldName: "Price", Header: "Price", Width: 10},
				{FieldName: "Features_Color", Header: "Color", Width: 15},
			},
		}).
		Build()

	// 3. Export to temp file
	tmpFile := "test_dynamic_export.xlsx"
	defer os.Remove(tmpFile)

	err := exporter.ExportToExcel(context.Background(), tmpFile)
	if err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// 4. Verify the file
	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer f.Close()

	// Verify headers (Row 2, assuming title is Row 1)
	headers := map[string]string{
		"A2": "Product Name",
		"B2": "Price",
		"C2": "Color",
	}
	for cell, expected := range headers {
		val, err := f.GetCellValue("DynamicSheet", cell)
		if err != nil {
			t.Fatalf("GetCellValue failed: %v", err)
		}
		if val != expected {
			t.Errorf("Cell %s: expected %s, got %s", cell, expected, val)
		}
	}

	// Verify Data (Row 3)
	row3 := map[string]string{
		"A3": "Product A",
		"B3": "100",
		"C3": "Red",
	}
	for cell, expected := range row3 {
		val, err := f.GetCellValue("DynamicSheet", cell)
		if err != nil {
			t.Fatalf("GetCellValue failed: %v", err)
		}
		if val != expected {
			t.Errorf("Cell %s: expected %s, got %s", cell, expected, val)
		}
	}

	// Verify Data (Row 4)
	row4 := map[string]string{
		"A4": "Product B",
		"B4": "200",
		"C4": "Blue",
	}
	for cell, expected := range row4 {
		val, err := f.GetCellValue("DynamicSheet", cell)
		if err != nil {
			t.Fatalf("GetCellValue failed: %v", err)
		}
		if val != expected {
			t.Errorf("Cell %s: expected %s, got %s", cell, expected, val)
		}
	}
}
