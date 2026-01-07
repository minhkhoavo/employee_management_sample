package simpleexcelv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigModification(t *testing.T) {
	yamlConfig := `
sheets:
  - name: "Employees"
    sections:
      - id: "emp_section"
        type: "full"
        show_header: true
        columns:
          - field_name: "Name"
            header: "Full Name"
            width: 20
          - field_name: "Age"
            header: "Age"
            width: 10
`
	exporter, err := NewExcelDataExporterFromYamlConfig(yamlConfig)
	assert.NoError(t, err)

	// 1. Get Section
	section := exporter.GetSection("emp_section")
	assert.NotNil(t, section)

	// 2. Get Column and Modify
	col := section.GetColumn("Name")
	assert.NotNil(t, col)
	assert.Equal(t, "Full Name", col.Header)
	assert.Equal(t, 20.0, col.Width)

	// Modify
	col.Header = "Employee Name"
	col.Width = 30.0

	// 3. Verify modification persisted
	// Re-fetch to be sure
	col2 := section.GetColumn("Name")
	assert.Equal(t, "Employee Name", col2.Header)
	assert.Equal(t, 30.0, col2.Width)

	// 4. Verify getting non-existent column
	assert.Nil(t, section.GetColumn("NonExistent"))

	// 5. Verify getting non-existent section
	assert.Nil(t, exporter.GetSection("non_existent_section"))
}
