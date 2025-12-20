package simpleexcelv2

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataExporter_ColumnHeight(t *testing.T) {
	exporter := NewExcelDataExporter()
	exporter.AddSheet("Height Test").
		AddSection(&SectionConfig{
			Title:        "Tall Section",
			TitleHeight:  50,
			HeaderHeight: 30,
			DataHeight:   20,
			ShowHeader:   true,
			Columns: []ColumnConfig{
				{FieldName: "Name", Header: "Name", Width: 20},
				{FieldName: "Value", Header: "Value", Width: 20, Height: 40}, // Individual height > DataHeight
			},
			Data: []map[string]interface{}{
				{"Name": "Item 1", "Value": 100},
				{"Name": "Item 2", "Value": 200},
			},
		})

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	// Save for manual inspection if needed
	tmpFile := "test_height.xlsx"
	err = f.SaveAs(tmpFile)
	require.NoError(t, err)
	defer os.Remove(tmpFile)

	// Verify Title Row Height (Row 1)
	h, err := f.GetRowHeight("Height Test", 1)
	assert.NoError(t, err)
	assert.Equal(t, float64(50), h)

	// Verify Header Row Height (Row 2)
	h, err = f.GetRowHeight("Height Test", 2)
	assert.NoError(t, err)
	assert.Equal(t, float64(30), h)

	// Verify Data Row Height (Row 3 and 4)
	// Column "Value" has height 40, which should override Section's DataHeight 20
	h, err = f.GetRowHeight("Height Test", 3)
	assert.NoError(t, err)
	assert.Equal(t, float64(40), h)

	h, err = f.GetRowHeight("Height Test", 4)
	assert.NoError(t, err)
	assert.Equal(t, float64(40), h)
}

func TestDataExporter_YamlHeight(t *testing.T) {
	yamlConfig := `
sheets:
  - name: "Yaml Height Test"
    sections:
    - id: "s1"
      title: "Tall Section"
      title_height: 60
      header_height: 40
      data_height: 25
      show_header: true
      columns:
        - field_name: "Name"
          header: "Name"
          width: 20
        - field_name: "Value"
          header: "Value"
          width: 20
          height: 45
`
	exporter, err := NewExcelDataExporterFromYamlConfig(yamlConfig)
	require.NoError(t, err)

	data := []map[string]interface{}{
		{"Name": "Item 1", "Value": 100},
	}
	exporter.BindSectionData("s1", data)

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	// Row 1: Title
	h, _ := f.GetRowHeight("Yaml Height Test", 1)
	assert.Equal(t, float64(60), h)

	// Row 2: Header
	h, _ = f.GetRowHeight("Yaml Height Test", 2)
	assert.Equal(t, float64(40), h)

	// Row 3: Data (Max of DataHeight 25 and Column Height 45)
	h, _ = f.GetRowHeight("Yaml Height Test", 3)
	assert.Equal(t, float64(45), h)
}
