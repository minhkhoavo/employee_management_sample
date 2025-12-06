package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/pgexcel"
)

func main() {
	ctx := context.Background()

	// Load environment configuration
	if err := config.LoadEnvConfig(); err != nil {
		panic(err)
	}

	// Initialize logging
	logger.InitLogging(config.DefaultEnvConfig.LOG_FILE_PATH)
	logger.InfoLog(ctx, "Environment variables loaded successfully")

	// Initialize database connection
	dbConfig := database.Config{
		Host:            config.DefaultEnvConfig.DB_HOST,
		Port:            config.DefaultEnvConfig.DB_PORT,
		User:            config.DefaultEnvConfig.DB_USER,
		Password:        config.DefaultEnvConfig.DB_PASSWORD,
		DBName:          config.DefaultEnvConfig.DB_NAME,
		SSLMode:         config.DefaultEnvConfig.DB_SSL_MODE,
		MaxOpenConns:    config.DefaultEnvConfig.DB_MAX_OPEN_CONNS,
		MaxIdleConns:    config.DefaultEnvConfig.DB_MAX_IDLE_CONNS,
		ConnMaxLifetime: config.DefaultEnvConfig.DB_CONN_MAX_LIFETIME,
	}

	db, err := database.NewPostgresDB(ctx, dbConfig)
	if err != nil {
		logger.ErrorLog(ctx, "Failed to initialize database: %v", err)
		panic(err)
	}
	defer db.Close()

	// Run both examples
	programmaticExport(ctx)
	yamlConfigExport(ctx)
}

// programmaticExport demonstrates the original programmatic approach
func programmaticExport(ctx context.Context) {
	fmt.Println("Example 1: Programmatic Section Configuration")

	type EmployeeForExcel struct {
		ID        int
		FirstName string
		LastName  string
		BirthDate string
		HireDate  string
		Gender    string
	}
	type DeptManagerForExcel struct {
		DeptNo   string
		EmpNo    int
		FromDate string
		ToDate   string
	}

	employees := []domain.Employee{
		{ID: 1, FirstName: "Alice", LastName: "Johnson", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
		{ID: 2, FirstName: "Bob", LastName: "Smith", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 3, FirstName: "Carol", LastName: "Williams", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
		{ID: 4, FirstName: "David", LastName: "Brown", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
	}
	excelExportExmployees := []EmployeeForExcel{}
	for _, employee := range employees {
		excelExportExmployees = append(excelExportExmployees, EmployeeForExcel{
			ID:        employee.ID,
			FirstName: employee.FirstName,
			LastName:  employee.LastName,
			BirthDate: employee.BirthDate.Format("01/02/2006"),
			HireDate:  employee.HireDate.Format("01/02/2006"),
			Gender:    employee.Gender,
		})
	}

	level2Employees := []domain.Employee{
		{ID: 1, FirstName: "Level 2 Employee 1", LastName: "Level 2 Employee 1", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 2, FirstName: "Level 2 Employee 2", LastName: "Level 2 Employee 2", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
		{ID: 3, FirstName: "Level 2 Employee 3", LastName: "Level 2 Employee 3", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 4, FirstName: "Level 2 Employee 4", LastName: "Level 2 Employee 4", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
	}
	excelExportLevel2Employees := []EmployeeForExcel{}
	for _, employee := range level2Employees {
		excelExportLevel2Employees = append(excelExportLevel2Employees, EmployeeForExcel{
			ID:        employee.ID,
			FirstName: employee.FirstName,
			LastName:  employee.LastName,
			BirthDate: employee.BirthDate.Format("2006-01-02"),
			HireDate:  employee.HireDate.Format("2006-01-02"),
			Gender:    employee.Gender,
		})
	}
	deptManagers := []domain.DeptManager{
		{
			DeptNo:   "D001",
			EmpNo:    1,
			FromDate: time.Now(),
			ToDate:   time.Now(),
		},
		{
			DeptNo:   "D002",
			EmpNo:    2,
			FromDate: time.Now(),
			ToDate:   time.Now(),
		},
		{
			DeptNo:   "D003",
			EmpNo:    3,
			FromDate: time.Now(),
			ToDate:   time.Now(),
		},
		{
			DeptNo:   "D004",
			EmpNo:    4,
			FromDate: time.Now(),
			ToDate:   time.Now(),
		},
	}
	excelExportDeptManagers := []DeptManagerForExcel{}
	for _, deptManager := range deptManagers {
		excelExportDeptManagers = append(excelExportDeptManagers, DeptManagerForExcel{
			EmpNo:    deptManager.EmpNo,
			DeptNo:   deptManager.DeptNo,
			FromDate: deptManager.FromDate.Format("2006-01-02"),
			ToDate:   deptManager.ToDate.Format("2006-01-02"),
		})
	}

	err := pgexcel.NewDataExporter().
		AddSheet("Report").
		AddSection(&pgexcel.SectionConfig{
			ID:         "employees",
			Title:      "Employees (Editable)",
			Data:       excelExportExmployees,
			Locked:     false,
			ShowHeader: true,
			Direction:  pgexcel.SectionDirectionHorizontal,
			TitleStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#1565C0"},
			},
			HeaderStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#1976D2"},
			},
			Columns: []pgexcel.ColumnConfig{
				{FieldName: "ID", Header: "Test Employee ID", Width: 12},
				{FieldName: "FirstName", Header: "Test First Name", Width: 20},
				{FieldName: "LastName", Header: "Test Last Name", Width: 20},
				{FieldName: "BirthDate", Header: "Test Birth Date", Width: 15},
				{FieldName: "HireDate", Header: "Test Hire Date", Width: 15},
				{FieldName: "Gender", Header: "Test Gender", Width: 10},
			},
		}).
		AddSection(&pgexcel.SectionConfig{
			ID:         "level2_employees",
			Title:      "Level 2 Employees (Read-Only)",
			Data:       excelExportLevel2Employees,
			Locked:     true,
			ShowHeader: true,
			Direction:  pgexcel.SectionDirectionHorizontal,
			TitleStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#2E7D32"},
			},
			HeaderStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#4CAF50"},
			},
			Columns: []pgexcel.ColumnConfig{
				{FieldName: "ID", Header: "Employee ID", Width: 12},
				{FieldName: "FirstName", Header: "First Name", Width: 20},
				{FieldName: "LastName", Header: "Last Name", Width: 20},
				{FieldName: "BirthDate", Header: "Birth Date", Width: 15},
				{FieldName: "HireDate", Header: "Hire Date", Width: 15},
				{FieldName: "Gender", Header: "Gender", Width: 10},
			},
		}).
		AddSection(&pgexcel.SectionConfig{
			ID:         "dept_managers",
			Title:      "Manager Employees (Read-Only)",
			Data:       excelExportDeptManagers,
			Locked:     true,
			ShowHeader: true,
			Position:   "G10",
			Direction:  pgexcel.SectionDirectionHorizontal,
			TitleStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#c07015ff"},
			},
			HeaderStyle: &pgexcel.StyleTemplate{
				Font: &pgexcel.FontTemplate{Bold: true, Color: "#FFFFFF"},
				Fill: &pgexcel.FillTemplate{Color: "#c07015ff"},
			},
			Columns: []pgexcel.ColumnConfig{
				{FieldName: "ID", Header: "Manager ID", Width: 12},
				{FieldName: "DeptNo", Header: "Department Number", Width: 20},
				{FieldName: "FromDate", Header: "From Date", Width: 20},
				{FieldName: "ToDate", Header: "To Date", Width: 15},
			},
		}).
		Build().ExportToFile(ctx, "programmatic_export_output.xlsx")

	if err != nil {
		log.Fatalf("Programmatic export failed: %v", err)
	}
	fmt.Println("  -> Created: programmatic_export_output.xlsx")
}

// yamlConfigExport demonstrates the new YAML configuration approach
func yamlConfigExport(ctx context.Context) {
	fmt.Println("Example 2: YAML Configuration with Data Binding")
	type EmployeeForExcel struct {
		ID        int
		FirstName string
		LastName  string
		BirthDate string
		HireDate  string
		Gender    string
	}

	employees := []domain.Employee{
		{ID: 1, FirstName: "Alice", LastName: "Johnson", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
		{ID: 2, FirstName: "Bob", LastName: "Smith", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 3, FirstName: "Carol", LastName: "Williams", BirthDate: time.Now(), HireDate: time.Now(), Gender: "F"},
		{ID: 4, FirstName: "David", LastName: "Brown", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
	}
	excelExportEmployees := []EmployeeForExcel{}
	for _, employee := range employees {
		excelExportEmployees = append(excelExportEmployees, EmployeeForExcel{
			ID:        employee.ID,
			FirstName: employee.FirstName,
			LastName:  employee.LastName,
			BirthDate: employee.BirthDate.Format("01/02/2006"),
			HireDate:  employee.HireDate.Format("01/02/2006"),
			Gender:    employee.Gender,
		})
	}

	managerEmployees := []domain.Employee{
		{ID: 1, FirstName: "Manager 1", LastName: "Manager 1", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 2, FirstName: "Manager 2", LastName: "Manager 2", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 3, FirstName: "Manager 3", LastName: "Manager 3", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
		{ID: 4, FirstName: "Manager 4", LastName: "Manager 4", BirthDate: time.Now(), HireDate: time.Now(), Gender: "M"},
	}
	excelExportManagerEmployees := []EmployeeForExcel{}
	for _, employee := range managerEmployees {
		excelExportManagerEmployees = append(excelExportManagerEmployees, EmployeeForExcel{
			ID:        employee.ID,
			FirstName: employee.FirstName,
			LastName:  employee.LastName,
			BirthDate: employee.BirthDate.Format("01/02/2006"),
			HireDate:  employee.HireDate.Format("01/02/2006"),
			Gender:    employee.Gender,
		})
	}

	// Load configuration from YAML file and bind data at runtime
	exporter, err := pgexcel.NewDataExporterFromYamlFile("report_config.yaml")
	if err != nil {
		log.Fatalf("Failed to load YAML config: %v", err)
	}

	err = exporter.
		BindSectionData("employees", excelExportEmployees).
		BindSectionData("managers", excelExportManagerEmployees).
		ExportToFile(ctx, "yaml_config_export_output.xlsx")

	if err != nil {
		log.Fatalf("YAML config export failed: %v", err)
	}
	fmt.Println("  -> Created: yaml_config_export_output.xlsx")
}
