package main

import (
	"context"

	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
)

func main() {
	ctx := context.Background()
	if err := config.LoadEnvConfig(); err != nil {
		panic(err)
	}

	logger.InitLogging(config.DefaultEnvConfig.LOG_FILE_PATH)
	logger.InfoLog(ctx, "Environment variables loaded successfully")

	sb := builder.NewSQLBuilder()
	sql, args, err := sb.Select("*").From("employees").Where("emp_no = ?", 1001).Where("gender = ?", "M").BuildSafe()
	if err != nil {
		panic(err)
	}
	logger.InfoLog(ctx, "SQL: %s, Args: %v", sql, args)
}
