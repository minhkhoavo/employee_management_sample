package simpleexcelv2

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDataExporter_AutoFilter(t *testing.T) {
	exporter := NewExcelDataExporter()
	exporter.AddSheet("Filter Test").
		AddSection(&SectionConfig{
			Title:      "Filtered Section",
			HasFilter:  true,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10},
				{FieldName: "Name", Header: "Name", Width: 20},
			},
			Data: []map[string]interface{}{
				{"ID": 1, "Name": "Alice"},
				{"ID": 2, "Name": "Bob"},
			},
		})

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	// Save for manual inspection if needed
	tmpFile := "test_autofilter.xlsx"
	err = f.SaveAs(tmpFile)
	require.NoError(t, err)
	defer os.Remove(tmpFile)
}

func TestDataExporter_AutoFilterWithHiddenFields(t *testing.T) {
	exporter := NewExcelDataExporter()
	exporter.AddSheet("Filter Hidden Test").
		AddSection(&SectionConfig{
			Title:      "Filtered Section",
			HasFilter:  true,
			ShowHeader: true,
			Columns: []ColumnConfig{
				{FieldName: "ID", Header: "ID", Width: 10, HiddenFieldName: "db_id"},
				{FieldName: "Name", Header: "Name", Width: 20},
			},
			Data: []map[string]interface{}{
				{"ID": 1, "Name": "Alice"},
			},
		})

	f, err := exporter.BuildExcel()
	require.NoError(t, err)

	tmpFile := "test_autofilter_hidden.xlsx"
	err = f.SaveAs(tmpFile)
	require.NoError(t, err)
	defer os.Remove(tmpFile)
}
