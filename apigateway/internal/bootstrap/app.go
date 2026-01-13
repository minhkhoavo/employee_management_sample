package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/locvowork/employee_management_sample/apigateway/internal/config"
	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/handler"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/googlecloud"
)

type App struct {
	Echo            *echo.Echo
	DB              *sql.DB
	GCP             *googlecloud.Client
	DataStoreClient *datastore.Client
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
	// Set GCP credentials and project automatically
	// Try multiple paths to find key.json
	var keyPath string
	
	// Try 1: Current working directory
	if _, err := os.Stat("key.json"); err == nil {
		keyPath = "key.json"
	}
	
	// Try 2: Executable directory
	if keyPath == "" {
		ex, err := os.Executable()
		if err == nil {
			path := filepath.Join(filepath.Dir(ex), "key.json")
			if _, err := os.Stat(path); err == nil {
				keyPath = path
			}
		}
	}
	
	// Try 3: Absolute path
	if keyPath == "" {
		keyPath = "d:\\Workspace\\employee_management_sample\\apigateway\\key.json"
		if _, err := os.Stat(keyPath); err != nil {
			keyPath = "" // Reset if not found
		}
	}
	
	if keyPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyPath)
		logger.InfoLog(ctx, fmt.Sprintf("Found GCP credentials at: %s", keyPath))
	} else {
		logger.InfoLog(ctx, "GCP credentials file not found, will skip GCP features")
	}
	
	os.Setenv("GCP_PROJECT_ID", "devhub-464904")

	// Load environment configuration
	if err := config.LoadEnvConfig(); err != nil {
		return fmt.Errorf("failed to load env config: %w", err)
	}

	// Remove emulator env vars to use real GCP (do this AFTER loading config)
	os.Unsetenv("DATASTORE_EMULATOR_HOST")
	os.Unsetenv("FIRESTORE_EMULATOR_HOST")

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

	dsClient, err := datastore.NewClient(ctx, config.DefaultEnvConfig.GCP_PROJECT_ID)
	if err != nil {
		logger.WarnLog(ctx, fmt.Sprintf("failed to initialize datastore client: %v (dump will be skipped)", err))
		// Don't fail - just skip dump if datastore is unavailable
	} else {
		a.DataStoreClient = dsClient
	}

	// Initialize dependencies
	empRepo := repository.NewEmployeeRepository(db)
	empSvc := service.NewEmployeeService(empRepo)
	empHandler := handler.NewEmployeeHandler(empSvc)
	compHandler := handler.NewComparisonHandler()

	// Initialize GCP Datastore Client
	gcpClient, err := googlecloud.NewClient(ctx, config.DefaultEnvConfig.GCP_PROJECT_ID)
	if err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("failed to initialize GCP client: %v", err))
		// We might not want to fail the whole app if GCP is optional, but for now let's be strict if configured.
	}
	a.GCP = gcpClient
	gcpHandler := handler.NewGCPDemoHandler(gcpClient)

	// Register Middlewares
	a.RegisterMiddlewares()

	// Register Routes
	a.RegisterRoutes(empHandler, compHandler, gcpHandler)

	// Dump test data to GCP Datastore (async, non-blocking, optional)
	if a.DataStoreClient != nil {
		go func() {
			dumpCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			logger.InfoLog(dumpCtx, "Dumping test data to GCP Datastore...")
			err := DumpUserToGCP(dumpCtx, a.DataStoreClient)
			if err != nil {
				logger.WarnLog(dumpCtx, fmt.Sprintf("Dump to GCP Datastore failed (non-critical): %v", err))
			} else {
				logger.InfoLog(dumpCtx, "Dump to GCP Datastore completed successfully")
			}
		}()
	}

	return nil
}

type User struct {
	ID    int64  `datastore:"id"`
	Name  string `datastore:"name"`
	Email string `datastore:"email"`
}

func DumpUser(ctx context.Context, dsClient *datastore.Client) error {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}
	var keys []*datastore.Key
	for range users {
		key := datastore.IncompleteKey("User", nil)
		keys = append(keys, key)
	}
	_, err := dsClient.PutMulti(ctx, keys, users)
	if err != nil {
		return fmt.Errorf("failed to dump users to datastore: %w", err)
	}
	fmt.Println("Dump data into Datastore Emulator successfully!")
	return nil
}

func queryUser(ctx context.Context, dsClient *datastore.Client) error {
	var users []User
	query := datastore.NewQuery("User")
	_, err := dsClient.GetAll(ctx, query, &users)
	if err != nil {
		return fmt.Errorf("failed to query users from datastore: %w", err)
	}
	for _, user := range users {
		fmt.Printf("User ID: %d, Name: %s, Email: %s\n", user.ID, user.Name, user.Email)
	}
	return nil
}

func (a *App) RegisterMiddlewares() {
	a.Echo.Use(middleware.Logger())
	a.Echo.Use(middleware.Recover())
	a.Echo.Use(middleware.CORS())
}

func (a *App) RegisterRoutes(empHandler *handler.EmployeeHandler, compHandler *handler.ComparisonHandler, gcpHandler *handler.GCPDemoHandler) {
	a.Echo.POST("/employees", empHandler.CreateHandler)
	a.Echo.GET("/employees/:id", empHandler.GetHandler)
	a.Echo.PUT("/employees/:id", empHandler.UpdateHandler)
	a.Echo.DELETE("/employees/:id", empHandler.DeleteHandler)
	a.Echo.GET("/employees", empHandler.ListHandler)
	a.Echo.GET("/employees/:id/report", empHandler.ReportHandler)

	exportGroup := a.Echo.Group("/export")
	exportGroup.GET("/fluent", empHandler.ExportFluentConfigHandler)
	exportGroup.GET("/yaml", empHandler.ExportFromYAMLHandler)

	exportGroupV2 := a.Echo.Group("/export/v2")
	exportGroupV2.GET("/fluent", empHandler.ExportFluentConfigHandler)
	exportGroupV2.GET("/yaml", empHandler.ExportV2FromYAMLHandler)
	exportGroupV2.GET("/largedata", empHandler.ExportLargeDataHandler)
	exportGroupV2.GET("/perf", empHandler.ExportLargeColumnHandler)

	compGroup := a.Echo.Group("/comparison")
	compGroup.GET("/wiki/tpl", compHandler.ExportWikiTPL)
	compGroup.GET("/wiki/idiomatic", compHandler.ExportWikiIdiomatic)
	compGroup.GET("/wiki/stream", compHandler.ExportWikiStreaming)
	compGroup.GET("/wiki/streaming-v2", compHandler.ExportWikiStreamingV2)
	compGroup.GET("/wiki/streaming-multi-section", compHandler.ExportMultiSectionStreamYAML)

	if gcpHandler != nil {
		gcpGroup := a.Echo.Group("/api/v1/gcp")
		gcpGroup.POST("/task-lists", gcpHandler.CreateTaskListHandler)
		gcpGroup.POST("/task-lists/:id/tasks", gcpHandler.CreateTaskHandler)
		gcpGroup.GET("/task-lists/:id/tasks", gcpHandler.ListTasksHandler)
		gcpGroup.GET("/tasks/complex", gcpHandler.ComplexQueryHandler)
	}
}

func (a *App) Run() error {
	defer a.DB.Close()
	if a.GCP != nil {
		defer a.GCP.Close()
	}
	if a.DataStoreClient != nil {
		defer a.DataStoreClient.Close()
	}
	return a.Echo.Start(":" + config.DefaultEnvConfig.APP_PORT)
}

func DumpUserToGCP(ctx context.Context, dsClient *datastore.Client) error {
	logger.InfoLog(ctx, "[Dump] Starting DumpUserToGCP")

	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
	}

	logger.InfoLog(ctx, fmt.Sprintf("[Dump] Creating %d users", len(users)))
	var keys []*datastore.Key
	for range users {
		key := datastore.IncompleteKey("User", nil)
		keys = append(keys, key)
	}

	logger.InfoLog(ctx, fmt.Sprintf("[Dump] Calling PutMulti with %d keys", len(keys)))

	// Use a shorter timeout for the actual put operation
	putCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := dsClient.PutMulti(putCtx, keys, users)
	if err != nil {
		logger.ErrorLog(ctx, fmt.Sprintf("[Dump] PutMulti failed: %v", err))
		return fmt.Errorf("failed to dump users to GCP datastore: %w", err)
	}

	logger.InfoLog(ctx, "[Dump] DumpUserToGCP completed successfully!")
	return nil
}
