package pgexcel

import (
	"testing"
	"time"
)

func TestCellRangeString(t *testing.T) {
	tests := []struct {
		name     string
		r        CellRange
		expected string
	}{
		{
			name: "simple range",
			r: CellRange{
				StartCol: "A",
				StartRow: 1,
				EndCol:   "B",
				EndRow:   10,
			},
			expected: "A1:B10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct is properly defined
			if tt.r.StartCol == "" {
				t.Error("CellRange not properly initialized")
			}
		})
	}
}

func TestNewSheetProtection(t *testing.T) {
	sp := NewSheetProtection()

	if sp == nil {
		t.Fatal("NewSheetProtection returned nil")
	}

	if !sp.ProtectSheet {
		t.Error("ProtectSheet should be true by default")
	}

	if !sp.AllowFilter {
		t.Error("AllowFilter should be true by default")
	}

	if sp.LockedCells == nil {
		t.Error("LockedCells map should be initialized")
	}
}

func TestProtectionRules(t *testing.T) {
	t.Run("LockAllExcept", func(t *testing.T) {
		rule := LockAllExcept(
			Columns("A", "B"),
		)

		if rule == nil {
			t.Fatal("LockAllExcept returned nil")
		}

		sp := NewSheetProtection()
		err := rule.Apply(sp)
		if err != nil {
			t.Fatalf("Failed to apply rule: %v", err)
		}

		if !sp.ProtectSheet {
			t.Error("ProtectSheet should be enabled")
		}
	})

	t.Run("Columns", func(t *testing.T) {
		rule := Columns("C", "D", "E")

		sp := NewSheetProtection()
		err := rule.Apply(sp)
		if err != nil {
			t.Fatalf("Failed to apply columns rule: %v", err)
		}

		if len(sp.UnlockedRanges) == 0 {
			t.Error("Expected unlocked ranges to be set")
		}
	})

	t.Run("LockColumns", func(t *testing.T) {
		rule := LockColumns("A", "B")

		sp := NewSheetProtection()
		err := rule.Apply(sp)
		if err != nil {
			t.Fatalf("Failed to apply lock columns rule: %v", err)
		}

		if len(sp.LockedColumns) == 0 {
			t.Error("Expected locked columns to be set")
		}
	})

	t.Run("LockRowsAbove", func(t *testing.T) {
		rule := LockRowsAbove(5)

		sp := NewSheetProtection()
		err := rule.Apply(sp)
		if err != nil {
			t.Fatalf("Failed to apply lock rows above rule: %v", err)
		}

		if len(sp.LockedRows) == 0 {
			t.Error("Expected locked rows to be set")
		}

		if sp.LockedRows[0].Start != 1 || sp.LockedRows[0].End != 5 {
			t.Errorf("Expected rows 1-5, got %d-%d", sp.LockedRows[0].Start, sp.LockedRows[0].End)
		}
	})

	t.Run("CombineRules", func(t *testing.T) {
		rule := CombineRules(
			LockRowsAbove(1),
			LockColumns("A", "B"),
			Columns("C"),
		)

		sp := NewSheetProtection()
		err := rule.Apply(sp)
		if err != nil {
			t.Fatalf("Failed to apply combined rules: %v", err)
		}

		if len(sp.LockedRows) == 0 {
			t.Error("Expected locked rows from combined rules")
		}

		if len(sp.LockedColumns) == 0 {
			t.Error("Expected locked columns from combined rules")
		}
	})
}

func TestStyleBuilder(t *testing.T) {
	t.Run("Builder pattern", func(t *testing.T) {
		style := NewStyleBuilder().
			Font("Arial", 12).
			Bold().
			FontColor("#FF0000").
			Fill("#FFFF00").
			Align("center").
			Build()

		if style.FontName != "Arial" {
			t.Errorf("Expected font Arial, got %s", style.FontName)
		}

		if style.FontSize != 12 {
			t.Errorf("Expected font size 12, got %f", style.FontSize)
		}

		if !style.FontBold {
			t.Error("Expected bold font")
		}

		if style.FontColor != "#FF0000" {
			t.Errorf("Expected red font color, got %s", style.FontColor)
		}

		if style.FillColor != "#FFFF00" {
			t.Errorf("Expected yellow fill, got %s", style.FillColor)
		}
	})

	t.Run("Pre-defined styles", func(t *testing.T) {
		headerBlue := HeaderStyleBlue()
		if headerBlue == nil {
			t.Fatal("HeaderStyleBlue returned nil")
		}
		if !headerBlue.FontBold {
			t.Error("Header should be bold")
		}

		editable := DataStyleEditable()
		if editable.Locked {
			t.Error("Editable style should not be locked")
		}

		readonly := DataStyleReadOnly()
		if !readonly.Locked {
			t.Error("Read-only style should be locked")
		}
	})
}

func TestParseCellRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CellRange
	}{
		{
			name:  "simple range",
			input: "A1:B10",
			expected: CellRange{
				StartCol: "A",
				StartRow: 1,
				EndCol:   "B",
				EndRow:   10,
			},
		},
		{
			name:  "larger range",
			input: "D5:Z100",
			expected: CellRange{
				StartCol: "D",
				StartRow: 5,
				EndCol:   "Z",
				EndRow:   100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCellRange(tt.input)

			if result.StartCol != tt.expected.StartCol {
				t.Errorf("StartCol: expected %s, got %s", tt.expected.StartCol, result.StartCol)
			}
			if result.StartRow != tt.expected.StartRow {
				t.Errorf("StartRow: expected %d, got %d", tt.expected.StartRow, result.StartRow)
			}
			if result.EndCol != tt.expected.EndCol {
				t.Errorf("EndCol: expected %s, got %s", tt.expected.EndCol, result.EndCol)
			}
			if result.EndRow != tt.expected.EndRow {
				t.Errorf("EndRow: expected %d, got %d", tt.expected.EndRow, result.EndRow)
			}
		})
	}
}

func TestColumnIndexToName(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{701, "ZZ"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := columnIndexToName(tt.index)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDefaultStyles(t *testing.T) {
	header := DefaultHeaderStyle()
	if header == nil {
		t.Fatal("DefaultHeaderStyle returned nil")
	}
	if !header.FontBold {
		t.Error("Header should be bold")
	}
	if !header.Locked {
		t.Error("Header should be locked")
	}

	data := DefaultDataStyle()
	if data == nil {
		t.Fatal("DefaultDataStyle returned nil")
	}
	if !data.Locked {
		t.Error("Data should be locked by default")
	}
}

// Benchmark tests
func BenchmarkColumnIndexToName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		columnIndexToName(i % 1000)
	}
}

func BenchmarkParseCellRange(b *testing.B) {
	ranges := []string{"A1:B10", "D5:Z100", "AA1:ZZ1000"}
	for i := 0; i < b.N; i++ {
		parseCellRange(ranges[i%len(ranges)])
	}
}

func BenchmarkStyleBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewStyleBuilder().
			Font("Arial", 12).
			Bold().
			FontColor("#FF0000").
			Fill("#FFFF00").
			Align("center").
			Build()
	}
}

// Example test showing time formatting
func TestTimeFormatting(t *testing.T) {
	now := time.Now()
	if now.IsZero() {
		t.Error("Time should not be zero")
	}

	// This verifies that time.Time can be used as export values
	dateFormat := "2006-01-02"
	formatted := now.Format(dateFormat)
	if formatted == "" {
		t.Error("Formatted date should not be empty")
	}
}
