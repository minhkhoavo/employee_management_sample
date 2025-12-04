package pgexcel

import (
	"strings"
	"testing"
)

func TestLoadTemplateFromString(t *testing.T) {
	yamlContent := `
version: "1.0"
name: "Test Report"
sheets:
  - name: "Sheet1"
    query: "SELECT * FROM test"
    columns:
      - name: "id"
        header: "ID"
`

	tmpl, err := LoadTemplateFromString(yamlContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if tmpl.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", tmpl.Version)
	}

	if tmpl.Name != "Test Report" {
		t.Errorf("Expected name 'Test Report', got %s", tmpl.Name)
	}

	if len(tmpl.Sheets) != 1 {
		t.Fatalf("Expected 1 sheet, got %d", len(tmpl.Sheets))
	}

	if tmpl.Sheets[0].Name != "Sheet1" {
		t.Errorf("Expected sheet name 'Sheet1', got %s", tmpl.Sheets[0].Name)
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty template",
			yaml:        `{}`,
			expectError: true,
			errorMsg:    "at least one sheet",
		},
		{
			name: "missing sheet name",
			yaml: `
sheets:
  - query: "SELECT * FROM test"
`,
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing query and query_file",
			yaml: `
sheets:
  - name: "Sheet1"
`,
			expectError: true,
			errorMsg:    "either query or query_file",
		},
		{
			name: "both query and query_file",
			yaml: `
sheets:
  - name: "Sheet1"
    query: "SELECT 1"
    query_file: "test.sql"
`,
			expectError: true,
			errorMsg:    "cannot specify both",
		},
		{
			name: "valid minimal template",
			yaml: `
sheets:
  - name: "Sheet1"
    query: "SELECT * FROM test"
`,
			expectError: false,
		},
		{
			name: "duplicate column names",
			yaml: `
sheets:
  - name: "Sheet1"
    query: "SELECT * FROM test"
    columns:
      - name: "id"
      - name: "id"
`,
			expectError: true,
			errorMsg:    "duplicate column name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadTemplateFromString(tt.yaml)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestColumnTemplateGetHeader(t *testing.T) {
	tests := []struct {
		name     string
		col      ColumnTemplate
		expected string
	}{
		{
			name:     "with header",
			col:      ColumnTemplate{Name: "col1", Header: "Column 1"},
			expected: "Column 1",
		},
		{
			name:     "without header",
			col:      ColumnTemplate{Name: "col1"},
			expected: "col1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.col.GetHeader(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestStyleTemplateToCellStyle(t *testing.T) {
	tests := []struct {
		name     string
		template *StyleTemplate
		check    func(*CellStyle) bool
	}{
		{
			name:     "nil template",
			template: nil,
			check:    func(s *CellStyle) bool { return s == nil },
		},
		{
			name: "font settings",
			template: &StyleTemplate{
				Font: &FontTemplate{
					Name:   "Arial",
					Size:   12,
					Bold:   true,
					Italic: true,
					Color:  "#FF0000",
				},
			},
			check: func(s *CellStyle) bool {
				return s.FontName == "Arial" &&
					s.FontSize == 12 &&
					s.FontBold == true &&
					s.FontItalic == true &&
					s.FontColor == "#FF0000"
			},
		},
		{
			name: "fill settings",
			template: &StyleTemplate{
				Fill: &FillTemplate{
					Color: "#FFFF00",
				},
			},
			check: func(s *CellStyle) bool {
				return s.FillColor == "#FFFF00" && s.FillPattern == 1
			},
		},
		{
			name: "alignment settings",
			template: &StyleTemplate{
				Alignment:  "center",
				VAlignment: "middle",
				WrapText:   true,
			},
			check: func(s *CellStyle) bool {
				return s.Alignment == "center" &&
					s.VerticalAlign == "middle" &&
					s.WrapText == true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.template.ToCellStyle()
			if !tt.check(result) {
				t.Errorf("ToCellStyle check failed for %s", tt.name)
			}
		})
	}
}

func TestStyleTemplateMerge(t *testing.T) {
	base := &StyleTemplate{
		Font: &FontTemplate{
			Name: "Arial",
			Size: 10,
		},
		Alignment: "left",
	}

	override := &StyleTemplate{
		Font: &FontTemplate{
			Bold:  true,
			Color: "#FF0000",
		},
		Alignment: "center",
	}

	merged := base.Merge(override)

	if merged.Font.Name != "Arial" {
		t.Errorf("Expected font name 'Arial', got %s", merged.Font.Name)
	}
	if !merged.Font.Bold {
		t.Error("Expected font bold to be true")
	}
	if merged.Font.Color != "#FF0000" {
		t.Errorf("Expected font color '#FF0000', got %s", merged.Font.Color)
	}
	if merged.Alignment != "center" {
		t.Errorf("Expected alignment 'center', got %s", merged.Alignment)
	}
}

func TestResolveString(t *testing.T) {
	tests := []struct {
		input    string
		vars     map[string]string
		expected string
	}{
		{
			input:    "Hello ${NAME}",
			vars:     map[string]string{"NAME": "World"},
			expected: "Hello World",
		},
		{
			input:    "${A} and ${B}",
			vars:     map[string]string{"A": "foo", "B": "bar"},
			expected: "foo and bar",
		},
		{
			input:    "No variables here",
			vars:     map[string]string{"NAME": "World"},
			expected: "No variables here",
		},
		{
			input:    "${MISSING}",
			vars:     map[string]string{},
			expected: "${MISSING}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := resolveString(tt.input, tt.vars)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	e := &TemplateExporter{}

	tests := []struct {
		value     interface{}
		condition string
		expected  bool
	}{
		// Numeric comparisons
		{100, "> 50", true},
		{100, "> 150", false},
		{100, "< 150", true},
		{100, ">= 100", true},
		{100, "<= 100", true},
		{100, "== 100", true},
		{100, "!= 100", false},

		// String comparisons
		{"ACTIVE", "== 'ACTIVE'", true},
		{"INACTIVE", "== 'ACTIVE'", false},
		{"ACTIVE", "!= 'INACTIVE'", true},

		// Contains
		{"Hello World", "contains 'World'", true},
		{"Hello World", "contains 'foo'", false},

		// Edge cases
		{nil, "> 0", false},
		{100, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			result := e.evaluateCondition(tt.value, tt.condition)
			if result != tt.expected {
				t.Errorf("evaluateCondition(%v, %s) = %v, expected %v",
					tt.value, tt.condition, result, tt.expected)
			}
		})
	}
}

func TestIsValidCellRange(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"A1:B10", true},
		{"AA1:ZZ100", true},
		{"A1", false},
		{"A1:B", false},
		{"1:10", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidCellRange(tt.input)
			if result != tt.expected {
				t.Errorf("isValidCellRange(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidRowRange(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1", true},
		{"100", true},
		{"1-5", true},
		{"10-20", true},
		{"header", false}, // Special case handled elsewhere
		{"abc", false},
		{"1-", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isValidRowRange(tt.input)
			if result != tt.expected {
				t.Errorf("isValidRowRange(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProtectionTemplateToSheetProtection(t *testing.T) {
	pt := &ProtectionTemplate{
		Password:        "test123",
		LockSheet:       true,
		AllowFilter:     true,
		AllowSort:       true,
		UnlockedColumns: []string{"A", "B"},
	}

	sp := pt.ToSheetProtection()

	if sp == nil {
		t.Fatal("Expected non-nil SheetProtection")
	}

	if sp.Password != "test123" {
		t.Errorf("Expected password 'test123', got %s", sp.Password)
	}

	if !sp.AllowFilter {
		t.Error("Expected AllowFilter to be true")
	}

	if !sp.AllowSort {
		t.Error("Expected AllowSort to be true")
	}
}

func TestProtectionTemplateNilLockSheet(t *testing.T) {
	pt := &ProtectionTemplate{
		LockSheet: false,
	}

	sp := pt.ToSheetProtection()

	if sp != nil {
		t.Error("Expected nil SheetProtection when LockSheet is false")
	}
}

// Benchmark tests

func BenchmarkLoadTemplateFromString(b *testing.B) {
	yaml := `
version: "1.0"
name: "Benchmark Test"
sheets:
  - name: "Sheet1"
    query: "SELECT * FROM test"
    columns:
      - name: "id"
        header: "ID"
      - name: "name"
        header: "Name"
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadTemplateFromString(yaml)
	}
}

func BenchmarkEvaluateCondition(b *testing.B) {
	e := &TemplateExporter{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.evaluateCondition(100.0, "> 50")
	}
}
