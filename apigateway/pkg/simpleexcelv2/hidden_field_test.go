package simpleexcelv2

import (
	"testing"
)

func TestDataExporter_HiddenFieldName(t *testing.T) {
	exporter := NewExcelDataExporter()

	type Product struct {
		Name  string
		Price float64
	}

	data := []Product{
		{"Laptop", 1200.0},
		{"Mouse", 25.0},
	}

	exporter.AddSheet("HiddenFieldSheet").
		AddSection(&SectionConfig{
			Title:      "Product List",
			ShowHeader: true,
			Data:       data,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Product Name", HiddenFieldName: "db_product_name"},
				{FieldName: "Price", Header: "Product Price", HiddenFieldName: "db_product_price"},
			},
		})

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	// Verify Hidden Row
	// Logic:
	// Row 1: Title ("Product List")
	// Row 2: Hidden Fields ("db_product_name", "db_product_price") -> Should be hidden
	// Row 3: Header ("Product Name", "Product Price")
	// Row 4: Data 1
	// Row 5: Data 2

	sheetName := "HiddenFieldSheet"

	// Check Row 2 values
	valA2, _ := excelFile.GetCellValue(sheetName, "A2")
	if valA2 != "db_product_name" {
		t.Errorf("Expected A2 to be 'db_product_name', got '%s'", valA2)
	}

	valB2, _ := excelFile.GetCellValue(sheetName, "B2")
	if valB2 != "db_product_price" {
		t.Errorf("Expected B2 to be 'db_product_price', got '%s'", valB2)
	}

	// Check if Row 2 is hidden
	visible, err := excelFile.GetRowVisible(sheetName, 2)
	if err != nil {
		t.Fatalf("Failed to get row visibility: %v", err)
	}
	if visible {
		t.Errorf("Expected Row 2 to be hidden, but it is visible")
	}

	// Check Header Row (Row 3)
	valA3, _ := excelFile.GetCellValue(sheetName, "A3")
	if valA3 != "Product Name" {
		t.Errorf("Expected A3 to be 'Product Name', got '%s'", valA3)
	}
}
