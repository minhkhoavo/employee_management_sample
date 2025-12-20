package simpleexcelv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataExporter_Alignment(t *testing.T) {
	exporter := NewExcelDataExporter()
	exporter.AddSheet("Alignment Test").
		AddSection(&SectionConfig{
			Title:      "Centered Title",
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
			},
			Data: []map[string]interface{}{
				{"ID": 1},
			},
		})

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	// Verify Title Alignment (Row 1, Col A)
	styleID, err := f.GetCellStyle("Alignment Test", "A1")
	require.NoError(t, err)

	style, err := f.GetStyle(styleID)
	require.NoError(t, err)

	assert.NotNil(t, style.Alignment)
	assert.Equal(t, "center", style.Alignment.Horizontal)
	assert.Equal(t, "top", style.Alignment.Vertical)

	// Verify Header Alignment (Row 2, Col A)
	styleID, err = f.GetCellStyle("Alignment Test", "A2")
	require.NoError(t, err)

	style, err = f.GetStyle(styleID)
	require.NoError(t, err)

	assert.NotNil(t, style.Alignment)
	assert.Equal(t, "center", style.Alignment.Horizontal)
	assert.Equal(t, "top", style.Alignment.Vertical)
}

func TestDataExporter_CustomAlignment(t *testing.T) {
	yamlConfig := `
sheets:
  - name: "Custom Align"
    sections:
    - id: "s1"
      title: "Left Bottom Title"
      title_style:
        alignment:
          horizontal: "left"
          vertical: "bottom"
      columns:
        - field_name: "ID"
`
	exporter, err := NewExcelDataExporterFromYamlConfig(yamlConfig)
	require.NoError(t, err)

	exporter.BindSectionData("s1", []map[string]interface{}{{"ID": 1}})

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	styleID, err := f.GetCellStyle("Custom Align", "A1")
	style, _ := f.GetStyle(styleID)

	assert.Equal(t, "left", style.Alignment.Horizontal)
	assert.Equal(t, "bottom", style.Alignment.Vertical)
}
