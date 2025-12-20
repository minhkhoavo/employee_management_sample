package simpleexcelv2

import (
	"testing"
)

func TestDataExporter_DefaultUnlockedCells(t *testing.T) {
	exporter := NewExcelDataExporter()

	type Product struct {
		Name string
	}
	data := []Product{{"Item 1"}}

	exporter.AddSheet("ProtectionTest").
		AddSection(&SectionConfig{
			Title:      "Locked Section",
			Locked:     true, // This triggers sheet protection
			ShowHeader: true,
			Data:       data,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Name"},
			},
		})

	excelFile, err := exporter.BuildExcel()
	if err != nil {
		t.Fatalf("Failed to build excel: %v", err)
	}

	sheetName := "ProtectionTest"

	// 1. Verify that a cell in the locked section is LOCKED
	// Section is at top left.
	// Row 1: Title
	// Row 2: Header
	// Row 3: Data -> Should be locked

	styleIDLocked, err := excelFile.GetCellStyle(sheetName, "A3")
	if err != nil {
		t.Fatalf("Failed to get style for A3: %v", err)
	}
	styleLocked, _ := excelFile.GetStyle(styleIDLocked)
	if styleLocked.Protection == nil || !styleLocked.Protection.Locked {
		t.Errorf("Expected A3 (in locked section) to be locked")
	}

	// 2. Verify that a cell OUTSIDE the locked section is UNLOCKED
	// e.g., cell Z100
	styleIDUnlocked, err := excelFile.GetCellStyle(sheetName, "Z100")
	if err != nil {
		t.Logf("GetCellStyle for Z100 returned error (expected if no specific style set, but column style should apply): %v", err)
		// If GetCellStyle returns 0 or error, check column style
		colStyleID, err := excelFile.GetColStyle(sheetName, "Z")
		if err != nil || colStyleID == 0 {
			// If no col style, default is locked property true in protected sheet?
			// Actually, if we set columns A:XFD to unlocked style, GetColStyle should return it.
			t.Errorf("Expected column Z to have a style set (unlocked)")
		} else {
			styleUnlocked, _ := excelFile.GetStyle(colStyleID)
			if styleUnlocked.Protection != nil && styleUnlocked.Protection.Locked {
				t.Errorf("Expected column Z (outside locked section) to be unlocked, but it is locked")
			}
		}
	} else {
		// If cell has style
		styleUnlocked, _ := excelFile.GetStyle(styleIDUnlocked)
		if styleUnlocked.Protection != nil && styleUnlocked.Protection.Locked {
			t.Errorf("Expected Z100 to be unlocked")
		}
	}
}
