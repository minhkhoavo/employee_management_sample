package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service/serviceutils"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcel"
)

type EmployeeHandler struct {
	svc service.EmployeeService
}

func NewEmployeeHandler(svc service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

func (h *EmployeeHandler) CreateHandler(c echo.Context) error {
	var req domain.Employee
	if err := c.Bind(&req); err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid request body", err)
	}

	if err := h.svc.Create(c.Request().Context(), &req); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to create employee", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusCreated, "Employee created successfully", nil)
}

func (h *EmployeeHandler) GetHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid employee ID", err)
	}

	emp, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to get employee", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusOK, "Employee retrieved successfully", emp)
}

func (h *EmployeeHandler) UpdateHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid employee ID", err)
	}

	var req domain.Employee
	if err := c.Bind(&req); err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid request body", err)
	}
	req.ID = id

	if err := h.svc.Update(c.Request().Context(), &req); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to update employee", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusOK, "Employee updated successfully", nil)
}

func (h *EmployeeHandler) DeleteHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid employee ID", err)
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to delete employee", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusOK, "Employee deleted successfully", nil)
}

func (h *EmployeeHandler) ListHandler(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	filter := domain.EmployeeFilter{
		Limit:  limit,
		Offset: offset,
	}

	employees, err := h.svc.List(c.Request().Context(), filter)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to list employees", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusOK, "Employees listed successfully", employees)
}

func (h *EmployeeHandler) ReportHandler(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusBadRequest, "Invalid employee ID", err)
	}

	report, err := h.svc.GetReport(c.Request().Context(), id)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to generate report", err)
	}

	return serviceutils.ResponseSuccess(c, http.StatusOK, "Employee report generated successfully", report)
}

// =============================================================================
// Sample Data For Excel Export
// =============================================================================

type Product struct {
	Name      string
	Price     float64
	Category  string
	Available bool
	Weight    float64
	Color     string
	// MetaData  map[string]interface{}
}

type Sale struct {
	Month  string
	Amount float64
	Region string
	Rep    string
}

// For the YAML based example
type ReportEmployee struct {
	ID        int
	FirstName string
	LastName  string
	BirthDate string
	HireDate  string
	Gender    string
}

func (h *EmployeeHandler) ExportFluentConfigHandler(c echo.Context) error {
	sampleSales := []Sale{
		{"January", 5000.0, "East", "Alice"},
		{"February", 4500.0, "West", "Bob"},
	}
	type HiddenSaleData struct {
		HiddenFieldName  string
		HiddenFieldValue interface{}
	}
	// Explicitly hidden data for demonstration
	hiddenData := []HiddenSaleData{
		{"Region", "North"},
		{"Rep", "SecretAgent"},
	}

	exporter := simpleexcel.NewDataExporter()

	// Sheet 1
	sheet1 := exporter.AddSheet("Sales Report")

	// Section 1: Visible Sales Data
	sheet1.AddSection(&simpleexcel.SectionConfig{
		Title:      "Visible Sales Data",
		ShowHeader: true,
		Data:       sampleSales,
		Columns: []simpleexcel.ColumnConfig{
			{FieldName: "Month", Header: "Month", Width: 15, HiddenFieldName: "db_month"},
			{FieldName: "Region", Header: "Region", Width: 15, HiddenFieldName: "db_region"},
			{FieldName: "Rep", Header: "Sales Rep", Width: 20, HiddenFieldName: "db_rep"},
			{FieldName: "Amount", Header: "Sale Amount", Width: 15, HiddenFieldName: "db_amount"},
		},
	})

	// Section 2: Hidden Data
	sheet1.AddSection(&simpleexcel.SectionConfig{
		Title: "Hidden Data Section",
		Type:  simpleexcel.SectionTypeHidden,
		Data:  hiddenData,
	})

	data, err := exporter.ToBytes()
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to generate excel file", err)
	}

	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="fluent_report_with_hidden.xlsx"`)
	c.Response().Header().Set("Content-Transfer-Encoding", "binary")

	_, err = c.Response().Write(data)
	return err
}

func (h *EmployeeHandler) ExportFromYAMLHandler(c echo.Context) error {
	// YAML configuration
	yamlConfig := ""
	data, err := os.ReadFile("report_config.yaml")
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
	type HiddenSaleData struct {
		HiddenFieldName  string
		HiddenFieldValue interface{}
	}
	// Explicitly hidden data for demonstration
	hiddenData := []HiddenSaleData{
		{"Region", "North"},
		{"Rep", "SecretAgent"},
	}

	// Convert Data
	dynamicDataEditable, err := simpleexcel.ConvertToDynamicData(productSectionEditable)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to convert dynamic data", err)
	}
	logger.InfoLog(c.Request().Context(), "Dynamic Data Editable: %+v", dynamicDataEditable)

	dynamicDataOriginal, err := simpleexcel.ConvertToDynamicData(productSectionOriginal)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to convert dynamic data", err)
	}
	logger.InfoLog(c.Request().Context(), "Dynamic Data Original: %+v", dynamicDataOriginal)

	// Initialize exporter with inline config
	exporter, err := simpleexcel.NewDataExporterFromYamlConfig(yamlConfig)
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
		BindSectionData("product_section_editable", dynamicDataEditable).
		BindSectionData("product_section_original", dynamicDataOriginal)

	// Demonstrate Mixed Config: Add a hidden section programmatically to the existing sheet
	if sheet := exporter.GetSheet("Executive Report"); sheet != nil {
		sheet.AddSection(&simpleexcel.SectionConfig{
			Title:      "Additional Hidden Data",
			Type:       simpleexcel.SectionTypeHidden,
			Data:       hiddenData,
			ShowHeader: true,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "HiddenFieldName", Header: "Field Name", Width: 20},
				{FieldName: "HiddenFieldValue", Header: "Field Value", Width: 20},
			},
		})
	}

	// Export to bytes
	excelBytes, err := exporter.ToBytes()
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to generate Excel file", err)
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="executive_report.xlsx"`)
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(excelBytes)))

	// Write response
	_, err = c.Response().Write(excelBytes)
	return err
}
