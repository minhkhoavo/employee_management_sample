package main

import (
	"context"

	"github.com/locvowork/employee_management_sample/apigateway/internal/bootstrap"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
)

func main() {
	ctx := context.Background()
	app := bootstrap.NewApp()
	if err := app.Initialize(ctx); err != nil {
		logger.ErrorLog(ctx, "Failed to initialize application: %v", err)
		panic(err)
	}

	if err := app.Run(); err != nil {
		logger.ErrorLog(ctx, "Application failed: %v", err)
		panic(err)
	}
}
