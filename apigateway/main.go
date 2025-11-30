package main

import (
	"context"

	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
)

func main() {
	ctx := context.Background()
	if err := config.LoadEnvConfig(); err != nil {
		panic(err)
	}

	logger.InitLogging(config.DefaultEnvConfig.LOG_FILE_PATH)
	logger.InfoLog(ctx, "Environment variables loaded successfully")
}
