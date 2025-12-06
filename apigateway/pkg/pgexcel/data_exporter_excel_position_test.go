package pgexcel

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDataExporterWithExcelPositioning(t *testing.T) {
	type Employee struct {
		ID   int
		Name string
		Role string
	}

	type Department struct {
		ID   int
		Name string
	}

	employees := []Employee{
		{ID: 1, Name: "Alice", Role: "Developer"},
		{ID: 2, Name: "Bob", Role: "Designer"},
	}

	departments := []Department{
		{ID: 1, Name: "Engineering"},
		{ID: 2, Name: "Design"},
	}

	tests := []struct {
		name          string
		section1      *SectionConfig
		section2      *SectionConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "valid excel positions",
			section1: &SectionConfig{
				Title:    "Employees",
				Data:     employees,
				Position: "A1",
			},
			section2: &SectionConfig{
				Title:    "Departments",
				Data:     departments,
				Position: "D1", // Positioned to the right of employees
			},
			expectError: false,
		},
		{
			name: "vertical positioning",
			section1: &SectionConfig{
				Title:    "Employees",
				Data:     employees,
				Position: "A1",
			},
			section2: &SectionConfig{
				Title:    "More Employees",
				Data:     employees,
				Position: "A10", // Positioned below first section
			},
			expectError: false,
		},
		{
			name: "invalid position format",
			section1: &SectionConfig{
				Title:    "Employees",
				Data:     employees,
				Position: "A0", // Invalid row number (must be >= 1)
			},
			expectError:   true,
			errorContains: "invalid Excel position format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheetBuilder := NewDataExporter().
				AddSheet("TestSheet").
				AddSection(tt.section1)

			// Only add second section if it's provided
			if tt.section2 != nil {
				sheetBuilder = sheetBuilder.AddSection(tt.section2)
			}

			exporter := sheetBuilder.Build()

			var buf bytes.Buffer
			err := exporter.Export(context.Background(), &buf)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Export with Excel positioning failed: %v", err)
				}
				if buf.Len() == 0 {
					t.Error("Expected non-empty output")
				}
			}
		})
	}
}
