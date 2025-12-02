package main

import (
	"context"

	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
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

	logger.InfoLog(ctx, "Database connection established successfully")
	sqlB := builder.NewSQLBuilder()
	query, args := sqlB.Select("*").
		From("employees.employee").
		Where("hire_date > ?", "1999-12-01").
		Limit(10).
		Build()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		logger.ErrorLog(ctx, "Failed to prepare statement: %v", err)
		panic(err)
	}
	defer stmt.Close()

	res, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		logger.ErrorLog(ctx, "Failed to query employee: %v", err)
		panic(err)
	}
	defer res.Close()

	for res.Next() {
		var e domain.Employee
		if err := res.Scan(&e.ID, &e.BirthDate, &e.FirstName, &e.LastName, &e.Gender, &e.HireDate); err != nil {
			logger.ErrorLog(ctx, "Failed to scan employee: %v", err)
			panic(err)
		}
		logger.InfoLog(ctx, "Employee: %#v", e)
	}
}
