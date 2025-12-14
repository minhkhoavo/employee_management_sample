package simpleexcel

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestDataExporterWithNamedFormatter(t *testing.T) {
	type Product struct {
		Name  string
		Price float64
	}

	data := []Product{
		{"Laptop", 1200.50},
	}

	exporter := NewDataExporter()

	// Register the formatter
	exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
		if price, ok := v.(float64); ok {
			return fmt.Sprintf("$%.2f", price)
		}
		return v
	})

	exporter.AddSheet("Named Formatter Test").
		AddSection(&SectionConfig{
			Data:       data,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Product"},
				{
					FieldName: "Price",
					Header:    "Price (Formatted)",
					// Simulate loading from YAML where Formatter func is nil but FormatterName is set
					FormatterName: "currency",
				},
			},
		})

	outputFile := "named_formatter_test.xlsx"
	defer os.Remove(outputFile)

	err := exporter.ExportToExcel(context.Background(), outputFile)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	// Verification relies on no error
}
