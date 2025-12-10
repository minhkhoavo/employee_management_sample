package handler

import (
	"net/http"
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
	MetaData  map[string]interface{}
}

type Sale struct {
	Month  string
	Amount float64
	Region string
	Rep    string
}

var sampleProducts = []Product{
	{"Laptop", 1200.50, "Electronics", true, map[string]interface{}{"Feature01": "16GB RAM", "Feature02": "512GB SSD"}},
	{"Mouse", 25.00, "Electronics", true, map[string]interface{}{"Feature01": "Wireless", "Feature03": "RGB"}},
	{"Book", 15.99, "Stationery", true, nil},
	{"Desk Chair", 150.80, "Furniture", false, map[string]interface{}{"Feature02": "Ergonomic"}},
}

var sampleSales = []Sale{
	{"January", 5000.0, "East", "Alice"},
	{"February", 4500.0, "West", "Bob"},
	{"January", 6000.0, "West", "Alice"},
	{"March", 7200.0, "East", "Charlie"},
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

var sampleReportEmployees = []ReportEmployee{
	{101, "John", "Doe", "1990-05-15", "2020-01-10", "Male"},
	{102, "Jane", "Smith", "1992-08-21", "2019-07-22", "Female"},
}

var sampleReportManagers = []ReportEmployee{
	{201, "Peter", "Jones", "1985-03-12", "2015-02-01", "Male"},
}

// =============================================================================
// Excel Export Handlers
// =============================================================================

func (h *EmployeeHandler) ExportSimpleHandler(c echo.Context) error {
	exporter := simpleexcel.NewDataExporter()

	exporter.AddSheet("Products").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Product Catalog",
			ShowHeader: true,
			Data:       sampleProducts,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 25},
				{FieldName: "Category", Header: "Category", Width: 15},
				{FieldName: "Price", Header: "Unit Price", Width: 15},
				{FieldName: "Available", Header: "In Stock", Width: 10},
			},
		})

	return exporter.StreamToResponse(c.Response().Writer, "products.xlsx")
}

func (h *EmployeeHandler) ExportComplexHandler(c echo.Context) error {
	exporter := simpleexcel.NewDataExporter()

	// Sheet 1: Multiple sections, different styles
	sheet1 := exporter.AddSheet("Sales Report")

	// Section 1: Sales Data (Vertical Layout)
	sheet1.AddSection(&simpleexcel.SectionConfig{
		Title:      "Quarterly Sales Data",
		ShowHeader: true,
		Data:       sampleSales,
		TitleStyle: &simpleexcel.StyleTemplate{
			Font: &simpleexcel.FontTemplate{Bold: true, Color: "FFFFFF"},
			Fill: &simpleexcel.FillTemplate{Color: "4472C4"},
		},
		HeaderStyle: &simpleexcel.StyleTemplate{
			Font: &simpleexcel.FontTemplate{Bold: true},
			Fill: &simpleexcel.FillTemplate{Color: "D9E1F2"},
		},
		Columns: []simpleexcel.ColumnConfig{
			{FieldName: "Month", Header: "Month", Width: 15},
			{FieldName: "Region", Header: "Region", Width: 15},
			{FieldName: "Rep", Header: "Sales Rep", Width: 20},
			{FieldName: "Amount", Header: "Sale Amount", Width: 15},
		},
	})

	// Section 2: Summary (Horizontal Layout)
	sheet1.AddSection(&simpleexcel.SectionConfig{
		Title:     "Report Summary",
		Direction: simpleexcel.SectionDirectionHorizontal,
		Position:  "F2", // Start horizontally from cell F2
		Data: []map[string]interface{}{
			{"Total Sales": 22700.0, "Region": "All"},
		},
		ShowHeader: true,
		Columns: []simpleexcel.ColumnConfig{
			{FieldName: "Total Sales", Header: "Total Sales", Width: 15},
			{FieldName: "Region", Header: "Region Filter", Width: 15},
		},
	})

	// Section 3: A title-only section
	sheet1.AddSection(&simpleexcel.SectionConfig{
		Type:     simpleexcel.SectionTypeTitleOnly,
		Title:    "Generated on: 2025-12-09",
		ColSpan:  4,
		Position: "A8",
		TitleStyle: &simpleexcel.StyleTemplate{
			Font: &simpleexcel.FontTemplate{Bold: false, Color: "888888"},
		},
	})

	// Sheet 2: Another sheet with different data
	exporter.AddSheet("Inventory").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Current Inventory Status",
			ShowHeader: true,
			Data:       sampleProducts,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 30},
				{FieldName: "Available", Header: "Is Available", Width: 15},
			},
		})

	return exporter.StreamToResponse(c.Response().Writer, "complex_report.xlsx")
}

func (h *EmployeeHandler) ExportFromYAMLHandler(c echo.Context) error {
	exporter, err := simpleexcel.NewDataExporterFromYamlFile("report_config.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to load report template", err)
	}

	exporter.
		BindSectionData("employees", sampleReportEmployees).
		BindSectionData("managers", sampleReportManagers)

	return exporter.StreamToResponse(c.Response().Writer, "report_from_yaml.xlsx")
}

func (h *EmployeeHandler) ExportWithLockingHandler(c echo.Context) error {
	exporter := simpleexcel.NewDataExporter()

	// By default, a section is not locked.
	// To make a sheet protected, at least one cell must be locked.
	// Unlocked cells will be editable, locked cells will be read-only.

	// Let''s say we want to lock the entire sheet except for the "Region" column.

	// We achieve this by:
	// 1. Setting `Locked: true` on the section, which makes all columns locked by default.
	// 2. Setting `Locked: false` on the specific column we want to be editable.

	isEditable := false // The desired state for the column

	exporter.AddSheet("Sales Data (Region Editable)").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Sales Data (Region is editable, other columns are read-only)",
			ShowHeader: true,
			Locked:     true, // 1. Lock the whole section by default
			Data:       sampleSales,
			Columns: []simpleexcel.ColumnConfig{
				{FieldName: "Month", Header: "Month", Width: 15},
				{
					FieldName: "Region",
					Header:    "Region (Editable)",
					Width:     20,
					Locked:    &isEditable, // 2. Override lock for this column
				},
				{FieldName: "Rep", Header: "Sales Rep", Width: 20},
				{FieldName: "Amount", Header: "Sale Amount", Width: 15},
			},
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true, Color: "FFFFFF"},
				Fill: &simpleexcel.FillTemplate{Color: "C00000"},
			},
			HeaderStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true},
			},
		})

	return exporter.StreamToResponse(c.Response().Writer, "locked_report.xlsx")
}

func (h *EmployeeHandler) ExportDynamicHandler(c echo.Context) error {
	// Original Column Configs
	// We set MetaData to be locked.
	locked := true
	cols := []simpleexcel.ColumnConfig{
		{FieldName: "Name", Header: "Product Name", Width: 20},
		{FieldName: "Price", Header: "Price", Width: 10},
		{FieldName: "Category", Header: "Category", Width: 15},
		{FieldName: "Available", Header: "In Stock", Width: 10},
		{
			FieldName: "MetaData",
			Header:    "Meta Data",
			Width:     15,
			Locked:    &locked, // This config should be inherited by dynamic fields
		},
	}

	// Convert Data
	dynamicData, newFields, err := simpleexcel.ConvertStructsToDynamic(sampleProducts, "MetaData")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to convert dynamic data", err)
	}
	logger.InfoLog(c.Request().Context(), "Dynamic Data: %+v", dynamicData)
	// Expand Columns
	// This will replace "MetaData" column with "Feature01", "Feature02", etc.
	// inheriting Locked=true and Width=15.
	finalCols := simpleexcel.ExpandColumnConfigs(cols, "MetaData", newFields)

	exporter := simpleexcel.NewDataExporter()
	exporter.AddSheet("Dynamic Products").
		AddSection(&simpleexcel.SectionConfig{
			Title:      "Dynamic Product Catalog",
			ShowHeader: true,
			Data:       dynamicData,
			Columns:    finalCols,
			TitleStyle: &simpleexcel.StyleTemplate{
				Font: &simpleexcel.FontTemplate{Bold: true},
			},
		})

	return exporter.StreamToResponse(c.Response().Writer, "dynamic_products.xlsx")
}

func (h *EmployeeHandler) ExportDynamicYamlHandler(c echo.Context) error {
	exporter, err := simpleexcel.NewDataExporterFromYamlFile("dynamic_report_config.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to load report template", err)
	}

	// Use BindDynamicSectionData to handle conversion and column expansion
	// Corresponds to section ID "dynamic_products_section" and map field "MetaData" in config
	if _, err := exporter.BindDynamicSectionData("dynamic_products_section", sampleProducts, "MetaData"); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to bind dynamic data", err)
	}

	return exporter.StreamToResponse(c.Response().Writer, "dynamic_products_yaml.xlsx")
}

func (h *EmployeeHandler) ExportDynamicHorizontalHandler(c echo.Context) error {
	exporter, err := simpleexcel.NewDataExporterFromYamlFile("dynamic_report_horizontal_config.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to load report template", err)
	}

	// 1. Bind Dynamic Data for the first section
	if _, err := exporter.BindDynamicSectionData("dynamic_products_section", sampleProducts, "MetaData"); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to bind dynamic data", err)
	}

	// 2. Bind Static Data for the second section
	exporter.BindSectionData("sales_section", sampleSales)

	return exporter.StreamToResponse(c.Response().Writer, "dynamic_horizontal_products.xlsx")
}
