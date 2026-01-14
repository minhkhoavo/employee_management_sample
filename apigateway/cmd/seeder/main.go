package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/locvowork/employee_management_sample/apigateway/internal/bootstrap"
	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
)

func main() {
	// Define flags
	action := flag.String("action", "seed", "Action to perform: seed, clear")
	preset := flag.String("preset", "large", "Data preset: small, medium, large, xlarge")
	brands := flag.Int("brands", 0, "Number of brands (overrides preset)")
	products := flag.Int("products", 0, "Number of products per brand (overrides preset)")
	features := flag.Int("features", 0, "Number of features per product (overrides preset)")

	flag.Parse()

	ctx := context.Background()

	fmt.Println("🚀 Product Data Seeder")
	fmt.Println(strings.Repeat("=", 50))

	// Initialize app
	fmt.Println("📡 Initializing application...")
	app := bootstrap.NewApp()
	if err := app.Initialize(ctx); err != nil {
		logger.ErrorLog(ctx, "Failed to initialize application: %v", err)
		log.Fatal(err)
	}

	// Get database connection
	db := app.DB
	if db == nil {
		logger.ErrorLog(ctx, "Database connection is nil")
		log.Fatal("Database connection is nil")
	}

	// Get Datastore client and wrap it
	dsRawClient := app.DataStoreClient
	if dsRawClient == nil {
		logger.ErrorLog(ctx, "Raw datastore client is nil")
		log.Fatal("Datastore client is nil")
	}

	dsClient := database.WrapDatastoreClient(dsRawClient)

	// Create seeder
	seeder := database.NewDataSeeder(db, dsClient)

	// Execute action
	switch *action {
	case "seed":
		performSeed(ctx, seeder, preset, brands, products, features)

	case "clear":
		performClear(ctx, seeder)

	default:
		fmt.Printf("❌ Unknown action: %s\n", *action)
		flag.PrintDefaults()
	}

	fmt.Println("\n✅ Done!")
}

func performSeed(ctx context.Context, seeder *database.DataSeeder, preset *string, brands, products, features *int) {
	var numBrands, numProducts, numFeatures int

	// Determine configuration
	if *brands > 0 && *products > 0 && *features > 0 {
		// Use custom values
		numBrands = *brands
		numProducts = *products
		numFeatures = *features
		fmt.Printf("📊 Using custom configuration: %d brands, %d products, %d features\n",
			numBrands, numProducts, numFeatures)
	} else {
		// Use preset
		presetConfig := database.SeedPreset(*preset)
		numBrands, numProducts, numFeatures = database.GetPresetConfig(presetConfig)
		fmt.Printf("📊 Using preset: %s\n", *preset)
	}

	// Perform seeding
	if err := seeder.SeedData(ctx, numBrands, numProducts, numFeatures); err != nil {
		log.Fatalf("❌ Seeding failed: %v", err)
	}
}

func performClear(ctx context.Context, seeder *database.DataSeeder) {
	fmt.Println("⚠️  This will delete all seeded data!")
	fmt.Print("Continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response == "yes" {
		if err := seeder.ClearData(ctx); err != nil {
			log.Fatalf("❌ Clear failed: %v", err)
		}
	} else {
		fmt.Println("Cancelled.")
	}
}
