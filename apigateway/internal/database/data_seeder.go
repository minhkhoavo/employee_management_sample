package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository"
)

type DataSeeder struct {
	db              *sql.DB
	datastoreClient *DatastoreClient
}

func NewDataSeeder(db *sql.DB, dc *DatastoreClient) *DataSeeder {
	return &DataSeeder{db: db, datastoreClient: dc}
}

var (
	brands       = []string{"Apple", "Samsung", "Sony", "LG", "Panasonic", "Philips", "Dell", "HP", "Lenovo", "ASUS"}
	countries    = []string{"USA", "China", "Vietnam", "Japan", "South Korea", "Germany", "Taiwan", "Thailand", "Malaysia", "Indonesia"}
	places       = []string{"New York", "Shanghai", "Hanoi", "Tokyo", "Seoul", "Berlin", "Taipei", "Bangkok", "Kuala Lumpur", "Jakarta"}
	featureNames = []string{"High Performance", "Energy Efficient", "Noise Reduction", "Smart Control", "Eco Friendly", "AI Powered", "Cloud Connected", "IoT Enabled", "Wireless", "USB-C"}
)

// SeedData đơn giản
func (ds *DataSeeder) SeedData(ctx context.Context, numBrands, numProductsPerBrand, numFeaturesPerCountry int) error {
	start := time.Now()
	fmt.Println("🚀 Seeding data...")

	rand.Seed(time.Now().UnixNano())

	if numBrands > len(brands) {
		numBrands = len(brands)
	}

	// 1. Tạo Products + Features (SQL)
	fmt.Println("📦 Creating products and features...")
	repo := repository.NewProductRepository(ds.db)

	var products []domain.Product
	var features []domain.Feature

	for b := 0; b < numBrands; b++ {
		brand := brands[b]

		for p := 1; p <= numProductsPerBrand; p++ {
			products = append(products, domain.Product{
				ID:       int64(p),
				Brand:    brand,
				Revision: 0,
			})

			// Mỗi product có features ở 2-5 countries
			numCountriesForProduct := rand.Intn(4) + 2
			selectedCountries := randomSelect(countries, numCountriesForProduct)

			for _, country := range selectedCountries {
				// Mỗi product-country có N features
				numFeatures := rand.Intn(numFeaturesPerCountry) + 1
				if numFeatures > 10 {
					numFeatures = 10
				}

				for i := 1; i <= numFeatures; i++ {
					features = append(features, domain.Feature{
						ID:        int64(p),
						Brand:     brand,
						Country:   country,
						Content:   featureNames[rand.Intn(len(featureNames))] + fmt.Sprintf(" v%d", i),
						SubNumber: i,
					})
				}
			}
		}
	}

	// Batch insert products
	if err := repo.BatchCreate(ctx, products); err != nil {
		return fmt.Errorf("failed to insert products: %w", err)
	}
	fmt.Printf("✅ Created %d products\n", len(products))

	// Batch insert features
	if err := ds.batchInsertFeatures(ctx, features); err != nil {
		return fmt.Errorf("failed to insert features: %w", err)
	}
	fmt.Printf("✅ Created %d features\n", len(features))

	// 2. Tạo ProductInfo (Datastore)
	fmt.Println("📋 Creating product infos in Datastore...")
	var productInfos []domain.ProductInfo

	for _, product := range products {
		numCountriesForProduct := rand.Intn(4) + 2
		selectedCountries := randomSelect(countries, numCountriesForProduct)

		for _, country := range selectedCountries {
			numProducts := rand.Intn(3) + 1
			for i := 1; i <= numProducts; i++ {
				productInfos = append(productInfos, domain.ProductInfo{
					ID:        product.ID,
					Brand:     product.Brand,
					Country:   country,
					Place:     places[rand.Intn(len(places))],
					Year:      2020 + rand.Intn(5),
					SubNumber: i,
				})
			}
		}
	}

	if err := ds.datastoreClient.BatchSaveProductInfos(ctx, productInfos); err != nil {
		return fmt.Errorf("failed to insert product infos: %w", err)
	}
	fmt.Printf("✅ Created %d product infos\n", len(productInfos))

	elapsed := time.Since(start)
	fmt.Printf("🎉 Done in %v\n", elapsed)
	fmt.Printf("📊 Stats: %d products, %d features, %d product infos\n", len(products), len(features), len(productInfos))

	return nil
}

func (ds *DataSeeder) batchInsertFeatures(ctx context.Context, feats []domain.Feature) error {
	if len(feats) == 0 {
		return nil
	}

	tx, err := ds.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO feature (id, brand, country, content, sub_number) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range feats {
		if _, err := stmt.ExecContext(ctx, f.ID, f.Brand, f.Country, f.Content, f.SubNumber); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (ds *DataSeeder) ClearData(ctx context.Context) error {
	fmt.Println("🗑️  Clearing data...")

	// Disable foreign key constraints temporarily
	if _, err := ds.db.ExecContext(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
		return fmt.Errorf("failed to defer constraints: %w", err)
	}

	// Clear features first (child table)
	if _, err := ds.db.ExecContext(ctx, "DELETE FROM feature"); err != nil {
		return fmt.Errorf("failed to delete features: %w", err)
	}

	// Clear products (parent table)
	if _, err := ds.db.ExecContext(ctx, "DELETE FROM product"); err != nil {
		return fmt.Errorf("failed to delete products: %w", err)
	}

	fmt.Println("✅ Cleared SQL data")
	return nil
}

// Presets
type SeedPreset string

const (
	PresetSmall  SeedPreset = "small"
	PresetMedium SeedPreset = "medium"
	PresetLarge  SeedPreset = "large"
	PresetXLarge SeedPreset = "xlarge"
)

// randomSelect randomly selects N items from a list
func randomSelect(items []string, count int) []string {
	if count > len(items) {
		count = len(items)
	}
	result := make([]string, count)
	perm := rand.Perm(len(items))
	for i := 0; i < count; i++ {
		result[i] = items[perm[i]]
	}
	return result
}

// GetPresetConfig returns configuration for a preset
func GetPresetConfig(preset SeedPreset) (numBrands, numProducts, numFeatures int) {
	switch preset {
	case PresetSmall:
		return 2, 10, 5
	case PresetMedium:
		return 5, 50, 10
	case PresetLarge:
		return 10, 100, 15
	case PresetXLarge:
		return 10, 500, 20
	default:
		return 5, 50, 10
	}
}
