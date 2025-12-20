package simpleexcelv2

import (
	"context"
	"os"
	"testing"
)

func TestDataExporterWithDefaultStyles(t *testing.T) {
	type Employee struct {
		ID   int
		Name string
	}

	data := []Employee{
		{1, "Alice"},
		{2, "Bob"},
	}

	exporter := NewExcelDataExporter()

	isLocked := true

	exporter.AddSheet("Style Defaults").
		AddSection(&SectionConfig{
			Data:       data,
			ShowHeader: true,
			// Section is not locked, but specific column is
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "ID"},
				{
					FieldName: "Name",
					Header:    "Name (Locked)",
					Locked:    &isLocked,
					// Should inherit default locked color (gray)
				},
			},
		})

	outputFile := "style_defaults_test.xlsx"
	defer os.Remove(outputFile)

	err := exporter.ExportToExcel(context.Background(), outputFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}
}
