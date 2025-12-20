package simpleexcelv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComparisonFeature(t *testing.T) {
	yamlConfig := `
sheets:
  - name: "Executive Report"
    sections:
    - id: "section_a"
      title: "Section A"
      direction: "horizontal"
      show_header: true
      columns:
        - field_name: "Name"
          header: "Name"
          hidden_field_name: "h_name"
        - field_name: "Value"
          header: "Value"
          hidden_field_name: "h_val"
    - id: "section_b"
      title: "Section B"
      direction: "horizontal"
      show_header: true
      columns:
        - field_name: "Name"
          header: "Name"
          hidden_field_name: "h_name"
        - field_name: "Value"
          header: "Value"
          hidden_field_name: "h_val"
    - id: "comparison"
      title: "Comparison"
      direction: "horizontal"
      show_header: true
      source_sections: ["section_a"]
      columns:
        - field_name: "Diff"
          header: "Diff Status"
          hidden_field_name: "h_diff"
          compare_with:
            section_id: "section_a"
            field_name: "Value"
          compare_against:
            section_id: "section_b"
            field_name: "Value"
`

	dataA := []map[string]interface{}{
		{"Name": "Item 1", "Value": 100},
		{"Name": "Item 2", "Value": 200},
	}
	dataB := []map[string]interface{}{
		{"Name": "Item 1", "Value": 100},
		{"Name": "Item 2", "Value": 250},
	}

	exporter, err := NewExcelDataExporterFromYamlConfig(yamlConfig)
	assert.NoError(t, err)

	exporter.BindSectionData("section_a", dataA)
	exporter.BindSectionData("section_b", dataB)

	f, err := exporter.BuildExcel()
	assert.NoError(t, err)

	// Section A: Col 1 (A), Title (1), Hidden (2), Header (3), Data (4-5)
	// Section B: Col 3 (C), Title (1), Hidden (2), Header (3), Data (4-5)
	// Comparison: Col 5 (E), Title (1), Hidden (2) - added, Header (3), Data (4-5) <- ALIGNED!

	// Cell E4: should have formula referencing Value in A and B at same row
	formula1, _ := f.GetCellFormula("Executive Report", "E4")
	assert.Equal(t, `IF(B4<>D4, "Diff", "")`, formula1)

	formula2, _ := f.GetCellFormula("Executive Report", "E5")
	assert.Equal(t, `IF(B5<>D5, "Diff", "")`, formula2)
}
