package simpleexcelv2

import (
	"bytes"
	"fmt"
	"testing"
)

func TestToWriter(t *testing.T) {
	// Create sample data
	type Item struct {
		ID   int
		Name string
	}
	data := make([]Item, 100)
	for i := 0; i < 100; i++ {
		data[i] = Item{ID: i + 1, Name: fmt.Sprintf("Item %d", i+1)}
	}

	exporter := NewExcelDataExporter().
		AddSheet("Test").
		AddSection(&SectionConfig{
			Title:      "Test Data",
			Data:       data,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "ID"},
				{FieldName: "Name", Header: "Name"},
			},
		}).
		Build()

	buf := new(bytes.Buffer)
	err := exporter.ToWriter(buf)
	if err != nil {
		t.Fatalf("ToWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("ToWriter produced empty output")
	}
}

func TestToCSV(t *testing.T) {
	// Create sample data
	type Item struct {
		ID   int
		Name string
	}
	data := []Item{
		{1, "Alice"},
		{2, "Bob"},
	}

	exporter := NewExcelDataExporter().
		AddSheet("Test").
		AddSection(&SectionConfig{
			Title:      "Users",
			Data:       data,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "User ID"},
				{FieldName: "Name", Header: "User Name"},
			},
		}).
		Build()

	buf := new(bytes.Buffer)
	err := exporter.ToCSV(buf)
	if err != nil {
		t.Fatalf("ToCSV failed: %v", err)
	}

	content := buf.String()
	// Expected CSV format (titles, headers, and rows often have empty lines between sections)
	expectedLines := []string{
		"Users",
		"User ID,User Name",
		"1,Alice",
		"2,Bob",
	}

	for _, line := range expectedLines {
		if !bytes.Contains(buf.Bytes(), []byte(line)) {
			t.Errorf("ToCSV output missing expected line: %s\nOutput:\n%s", line, content)
		}
	}
}
