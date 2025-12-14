package simpleexcel

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestDataExporter_MixedConfig(t *testing.T) {
	// 1. Create Exporter from YAML
	yamlConfig := `
sheets:
  - name: "MixedSheet"
    sections:
      - id: "sec1"
        title: "Section 1"
        type: "full"
        show_header: true
        columns:
          - field_name: "Col1"
            header: "Column 1"
`
	exporter, err := NewDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	// 2. Retrieve Sheet by Name
	sheet := exporter.GetSheet("MixedSheet")
	if sheet == nil {
		t.Fatalf("Failed to get sheet 'MixedSheet'")
	}

	// 3. Add Sections Programmatically
	sheet.AddSection(&SectionConfig{
		Title: "Programmatic Section",
		Data:  []struct{ ColA string }{{"Value A"}},
		Columns: []ColumnConfig{
			{FieldName: "ColA", Header: "Column A"},
		},
	})

	// 4. Bind Data for YAML section
	exporter.BindSectionData("sec1", []struct{ Col1 string }{{"Value 1"}})

	// 5. Build Excel
	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	// 6. Verify Content
	// Row 1: "Section 1" (Title)
	// Row 2: "Column 1" (Header)
	// Row 3: "Value 1" (Data)
	// Row 4: "Programmatic Section" (Title)
	// ...

	sheetName := "MixedSheet"
	valA1, _ := excelFile.GetCellValue(sheetName, "A1")
	if valA1 != "Section 1" {
		t.Errorf("Expected A1 to be 'Section 1', got '%s'", valA1)
	}

	// Since we don't know exact row index of second section easily without calculation logic (as it depends on gaps etc),
	// we assume standard vertical stacking.
	// Section 1 has 3 rows (Title, Header, Data).
	// Next section should start around Row 4 or 5 depending on gap.

	// Just check if we can find title "Programmatic Section" in column A
	found := false
	for i := 1; i <= 10; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		val, _ := excelFile.GetCellValue(sheetName, cell)
		if val == "Programmatic Section" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Could not find 'Programmatic Section' title in first 10 rows")
	}
}
