package pgexcel

import (
	"context"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLockedSectionHeaderStyle(t *testing.T) {
	// Define a simple struct for data
	type Item struct {
		Name string
	}
	data := []Item{{Name: "Item 1"}}

	// Create a section that is LOCKED and has a custom header color
	section := &SectionConfig{
		Title:  "Locked Section",
		Data:   data,
		Locked: true,
		HeaderStyle: &StyleTemplate{
			Fill: &FillTemplate{
				Color: "#FF0000", // Red background
			},
		},
	}

	// Export
	exporter := NewDataExporter().
		AddSheet("Sheet1").
		AddSection(section).
		Build()

	f := excelize.NewFile()
	// We need to access the internal export logic or just export to a buffer and read it back
	// But since we can't easily inspect styles from a buffer without saving/reopening,
	// let's use a temporary file.

	tmpFile := "test_locked_style.xlsx"
	err := exporter.ExportToFile(context.Background(), tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Open the file and check the style
	f, err = excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// The header should be at A2 (Title at A1)
	// Let's check the style of A2
	styleID, err := f.GetCellStyle("Sheet1", "A2")
	if err != nil {
		t.Fatalf("Failed to get cell style: %v", err)
	}

	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("Failed to get style details: %v", err)
	}

	// Check if fill is applied
	if style.Fill.Pattern != 1 || len(style.Fill.Color) == 0 || style.Fill.Color[0] != "FF0000" {
		t.Errorf("Expected header fill color FF0000, got %v", style.Fill)
	}

	// Check if locked
	if !style.Protection.Locked {
		t.Errorf("Expected cell to be locked")
	}
}
