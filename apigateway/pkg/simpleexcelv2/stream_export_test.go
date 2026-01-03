package simpleexcelv2

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestBatchWriting(t *testing.T) {
	// 1. Setup Exporter
	exporter := NewExcelDataExporter()
	sheet := exporter.AddSheet("StreamOps")

	// Title Section (Static)
	sheet.AddSection(&SectionConfig{
		Type:  SectionTypeTitleOnly,
		Title: "Batch Export Test",
	})

	// Data Section (Streaming)
	// We don't provide Data here, we will stream it.
	// ID is required to target the section during streaming.
	sheet.AddSection(&SectionConfig{
		ID:         "stream-data",
		ShowHeader: true,
		Columns: []ColumnConfig{
			{FieldName: "ID", Header: "ID", Width: 10},
			{FieldName: "Value", Header: "Value", Width: 30},
		},
	})

	// 2. Start Stream
	buf := new(bytes.Buffer)
	streamer, err := exporter.StartStream(buf)
	if err != nil {
		t.Fatalf("StartStream failed: %v", err)
	}

	// 3. Write Batch 1
	type DataItem struct {
		ID    int
		Value string
	}
	batch1 := []DataItem{
		{1, "A"},
		{2, "B"},
	}
	if err := streamer.Write("stream-data", batch1); err != nil {
		t.Fatalf("Write batch 1 failed: %v", err)
	}

	// 4. Write Batch 2
	batch2 := []DataItem{
		{3, "C"},
		{4, "D"},
	}
	if err := streamer.Write("stream-data", batch2); err != nil {
		t.Fatalf("Write batch 2 failed: %v", err)
	}

	// 5. Close
	if err := streamer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 6. Verify Content
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to open generated excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("StreamOps")
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	// Expected Rows:
	// Row 1: Batch Export Test (Title)
	// Row 2: ID, Value (Header)
	// Row 3: 1, A
	// Row 4: 2, B
	// Row 5: 3, C
	// Row 6: 4, D

	if len(rows) != 6 {
		t.Errorf("Expected 6 rows, got %d", len(rows))
	}

	if rows[0][0] != "Batch Export Test" {
		t.Errorf("Title incorrect, got %s", rows[0][0])
	}
	if rows[2][0] != "1" || rows[5][1] != "D" {
		t.Errorf("Data seemingly incorrect: %v", rows)
	}
}

func TestMultiSectionStreamYAML(t *testing.T) {
	// Note: This test requires github.com/stretchr/testify/assert for assert.NoError and assert.Equal
	// For simplicity, I'm assuming it's available or will be added.
	// If not, these should be replaced with standard Go testing.T methods.
	// For this response, I'll use t.Fatalf and t.Errorf for consistency with the existing file.

	yamlConfig := `
sheets:
  - name: "MultiStream"
    sections:
      - id: "part1"
        type: "full"
        title: "Part 1"
        show_header: true
        columns:
          - header: "ID1"
            field_name: "ID"
      
      - type: "title"
        title: "--- Middle ---"
        title_style:
          font: { bold: true }
      
      - id: "part2"
        type: "full"
        title: "Part 2"
        show_header: true
        columns:
          - header: "ID2"
            field_name: "ID"
`

	exporter, err := NewExcelDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		t.Fatalf("NewExcelDataExporterFromYamlConfig failed: %v", err)
	}

	var buf bytes.Buffer
	streamer, err := exporter.StartStream(&buf)
	if err != nil {
		t.Fatalf("StartStream failed: %v", err)
	}

	// Batch 1 -> Part 1
	data1 := []struct {
		ID int
	}{{ID: 101}, {ID: 102}}
	err = streamer.Write("part1", data1)
	if err != nil {
		t.Fatalf("Write to part1 failed: %v", err)
	}

	// Batch 2 -> Part 2
	data2 := []struct {
		ID int
	}{{ID: 201}, {ID: 202}}
	err = streamer.Write("part2", data2)
	if err != nil {
		t.Fatalf("Write to part2 failed: %v", err)
	}

	err = streamer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("Failed to open generated excel: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("MultiStream")
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	// Layout:
	// Row 1: Part 1 (Title)
	// Row 2: ID1 (Header)
	// Row 3: 101
	// Row 4: 102
	// Row 5: --- Middle --- (Title Section)
	// Row 6: Part 2 (Title)
	// Row 7: ID2 (Header)
	// Row 8: 201
	// Row 9: 202

	expectedRows := 9
	if len(rows) != expectedRows {
		t.Fatalf("Expected %d rows, got %d. Content: %v", expectedRows, len(rows), rows)
	}

	if rows[0][0] != "Part 1" {
		t.Errorf("Row 0, Col 0 expected 'Part 1', got '%s'", rows[0][0])
	}
	if rows[2][0] != "101" {
		t.Errorf("Row 2, Col 0 expected '101', got '%s'", rows[2][0])
	}
	if rows[4][0] != "--- Middle ---" {
		t.Errorf("Row 4, Col 0 expected '--- Middle ---', got '%s'", rows[4][0])
	}
	if rows[5][0] != "Part 2" {
		t.Errorf("Row 5, Col 0 expected 'Part 2', got '%s'", rows[5][0])
	}
	if rows[7][0] != "201" {
		t.Errorf("Row 7, Col 0 expected '201', got '%s'", rows[7][0])
	}
}
