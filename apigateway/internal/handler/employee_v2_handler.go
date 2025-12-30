package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service/serviceutils"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcelv2"
)

func (h *EmployeeHandler) ExportV2FromYAMLHandler(c echo.Context) error {
	// YAML configuration
	yamlConfig := ""
	data, err := os.ReadFile("report_config_v2.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to read YAML file", err)
	}
	yamlConfig = string(data)

	productSectionEditable := []Product{
		{
			Name:      "Laptop Pro",
			Price:     1299.99,
			Category:  "Electronics",
			Available: true,
			Weight:    2.5,
			Color:     "Silver",
		},
		{
			Name:      "Smartphone X",
			Price:     899.99,
			Category:  "Electronics",
			Available: false,
			Weight:    0.3,
			Color:     "Black",
		},
	}
	productSectionOriginal := []Product{
		{
			Name:      "Laptop Pro",
			Price:     1299.99,
			Category:  "Electronics",
			Available: true,
			Weight:    2.5,
			Color:     "Silver",
		},
		{
			Name:      "Smartphone X",
			Price:     899.99,
			Category:  "Electronics",
			Available: false,
			Weight:    0.3,
			Color:     "Black",
		},
	}

	// Initialize exporter with inline config
	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to parse inline report config", err)
	}

	// Register a simple currency formatter for demonstration
	exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
		if val, ok := v.(float64); ok {
			return fmt.Sprintf("$%.2f", val)
		}
		return v
	})

	// Bind data
	exporter.
		BindSectionData("product_section_editable", productSectionEditable).
		BindSectionData("product_section_original", productSectionOriginal)

	// Export to bytes
	excelBytes, err := exporter.ToBytes()
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to generate Excel file", err)
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="comparason_report.xlsx"`)
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(excelBytes)))

	// Write response
	_, err = c.Response().Write(excelBytes)
	return err
}

func (h *EmployeeHandler) ExportLargeDataHandler(c echo.Context) error {
	// Generate large dataset
	count := 50000
	data := make([]Product, count)
	for i := 0; i < count; i++ {
		data[i] = Product{
			Name:      fmt.Sprintf("Product %d", i+1),
			Price:     10.0 + float64(i)*0.01,
			Category:  "Bulk Item",
			Available: i%2 == 0,
			Weight:    1.0,
			Color:     "Generic",
		}
	}

	// Create and configure exporter
	exporter := simpleexcelv2.NewExcelDataExporter().
		AddSheet("Large Export").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      fmt.Sprintf("Bulk Products Export (%d rows)", count),
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 30},
				{FieldName: "Price", Header: "Unit Price", Width: 15},
				{FieldName: "Category", Header: "Category", Width: 20},
				{FieldName: "Available", Header: "In Stock", Width: 10},
			},
		}).
		Build()

	// Set headers for file download
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="large_products_%d.xlsx"`, count))

	// Stream directly to response
	return exporter.ToWriter(c.Response().Writer)
}
