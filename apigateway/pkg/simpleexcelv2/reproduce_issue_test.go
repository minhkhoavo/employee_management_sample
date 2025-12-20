package simpleexcelv2

import (
	"context"
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLockedStatus(t *testing.T) {
	// 1. Setup Data
	type TestData struct {
		Name string
	}
	data := []TestData{{Name: "Test"}}

	// 2. Create Exporter with Locked=false
	exporter := NewExcelDataExporter().
		AddSheet("Sheet1").
		AddSection(&SectionConfig{
			ID:         "test_section",
			Title:      "Test Section",
			Data:       data,
			Locked:     false, // Explicitly unlocked
			ShowHeader: true,
			Direction:  SectionDirectionHorizontal,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Name", Width: 20},
			},
		}).
		Build()

	// 3. Export to temp file
	tmpFile := "test_locked.xlsx"
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

	// Get style of cell A3 (Title is A1, Header is A2, Data is A3)
	// Wait, Title takes A1. Header takes A2. Data starts at A3.
	// Let's check A3.

	// Get cell style ID
	styleID, err := f.GetCellStyle("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellStyle failed: %v", err)
	}

	// Get style definition
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle failed: %v", err)
	}

	// Check Protection
	if style.Protection == nil {
		// If Protection is nil, it defaults to Locked=true in Excel?
		// Or maybe excelize returns nil if it's default?
		// Let's print it.
		t.Logf("Style Protection is nil")
	} else {
		t.Logf("Style Protection: Locked=%v", style.Protection.Locked)
		if style.Protection.Locked {
			t.Errorf("Expected Locked=false, got true")
		}
	}

	// Check Sheet Protection Options
	// Note: GetSheetProtection might not be available in all versions, but let's try.
	// If not available, we can't easily verify without inspecting XML, but let's assume it is.
	// Actually, let's just inspect the XML manually if this fails to compile.
	// But wait, we can't easily inspect XML in this environment without unzip.
	// Let's rely on the fact that we can modify the code to set it explicitly.

	// However, to be sure, let's try to verify.
	// We can use a workaround: try to set a value and see if it changes? No.

	// Let's just assume the hypothesis is strong and try to fix it in the plan.
	// But I should confirm.
	// I'll try to use GetSheetProtection.

}
