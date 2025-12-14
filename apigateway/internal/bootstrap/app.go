package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/handler"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
)

type App struct {
	Echo *echo.Echo
	DB   *sql.DB
	// `type envConfig struct` -> unexported.
	// I should probably export it if I want to put it in the struct, or just use `interface{}` or ignore it in the struct.
	// For now, I'll skip storing config in App struct if not strictly needed, or just use the global.
}

func NewApp() *App {
	return &App{
		Echo: echo.New(),
	}
}

func (a *App) Initialize(ctx context.Context) error {
	// Load environment configuration
	if err := config.LoadEnvConfig(); err != nil {
		return fmt.Errorf("failed to load env config: %w", err)
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
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	a.DB = db

	// Initialize dependencies
	empRepo := repository.NewEmployeeRepository(db)
	empSvc := service.NewEmployeeService(empRepo)
	empHandler := handler.NewEmployeeHandler(empSvc)

	// Register Middlewares
	a.RegisterMiddlewares()

	// Register Routes
	a.RegisterRoutes(empHandler)

	return nil
}

func (a *App) RegisterMiddlewares() {
	a.Echo.Use(middleware.Logger())
	a.Echo.Use(middleware.Recover())
	a.Echo.Use(middleware.CORS())
}

func (a *App) RegisterRoutes(empHandler *handler.EmployeeHandler) {
	a.Echo.POST("/employees", empHandler.CreateHandler)
	a.Echo.GET("/employees/:id", empHandler.GetHandler)
	a.Echo.PUT("/employees/:id", empHandler.UpdateHandler)
	a.Echo.DELETE("/employees/:id", empHandler.DeleteHandler)
	a.Echo.GET("/employees", empHandler.ListHandler)
	a.Echo.GET("/employees/:id/report", empHandler.ReportHandler)

	exportGroup := a.Echo.Group("/export")
	exportGroup.GET("/fluent", empHandler.ExportFluentConfigHandler)
	exportGroup.GET("/yaml", empHandler.ExportFromYAMLHandler)
}

func (a *App) Run() error {
	defer a.DB.Close()
	return a.Echo.Start(":" + config.DefaultEnvConfig.APP_PORT)
}
