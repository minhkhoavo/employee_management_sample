package simpleexcelv2

import (
	"fmt"
	"testing"
)

func BenchmarkRenderSections(b *testing.B) {
	// Create a reasonably large dataset
	type BenchRow struct {
		ID    int
		Name  string
		Value float64
		Note  string
	}

	rows := 1000 // 1000 rows per section for benchmark
	data := make([]BenchRow, rows)
	for i := 0; i < rows; i++ {
		data[i] = BenchRow{
			ID:    i,
			Name:  fmt.Sprintf("Item %d", i),
			Value: float64(i) * 1.5,
			Note:  "Some long note to simulate content",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exporter := NewExcelDataExporter()
		exporter.AddSheet("BenchSheet").
			AddSection(&SectionConfig{
				Title:      "Benchmark Section",
				Data:       data,
				ShowHeader: true,
				Columns: []ColumnConfig{
					{FieldName: "ID", Header: "ID", Width: 10},
					{FieldName: "Name", Header: "Name", Width: 30},
					{FieldName: "Value", Header: "Value", Width: 20},
					{FieldName: "Note", Header: "Notes", Width: 50},
				},
			})

		_, err := exporter.BuildExcel()
		if err != nil {
			b.Fatalf("BuildExcel failed: %v", err)
		}
	}
}
