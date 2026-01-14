package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
)

// FeatureRepository manages feature database operations
type FeatureRepository struct {
	db *sql.DB
}

// NewFeatureRepository creates a new repository
func NewFeatureRepository(db *sql.DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// GetAll retrieves all features
func (r *FeatureRepository) GetAll(ctx context.Context) ([]domain.Feature, error) {
	query := `
		SELECT id, brand, country, content, sub_number
		FROM feature
		ORDER BY brand, id, country, sub_number
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all features: %w", err)
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		err := rows.Scan(&f.ID, &f.Brand, &f.Country, &f.Content, &f.SubNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return features, nil
}

// GetByBrands retrieves features for specific brands
func (r *FeatureRepository) GetByBrands(ctx context.Context, brands []string) ([]domain.Feature, error) {
	if len(brands) == 0 {
		return []domain.Feature{}, nil
	}

	// Build placeholders for IN clause: ($1, $2, $3, ...)
	placeholders := make([]string, len(brands))
	args := make([]interface{}, len(brands))
	for i, brand := range brands {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = brand
	}

	query := fmt.Sprintf(`
		SELECT id, brand, country, content, sub_number
		FROM feature
		WHERE brand IN (%s)
		ORDER BY brand, id, country, sub_number
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get features by brands: %w", err)
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		err := rows.Scan(&f.ID, &f.Brand, &f.Country, &f.Content, &f.SubNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return features, nil
}

// GetByBrandAndID retrieves features for specific brand and product ID
func (r *FeatureRepository) GetByBrandAndID(ctx context.Context, brand string, id int64) ([]domain.Feature, error) {
	query := `
		SELECT id, brand, country, content, sub_number
		FROM feature
		WHERE brand = $1 AND id = $2
		ORDER BY country, sub_number
	`

	rows, err := r.db.QueryContext(ctx, query, brand, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		err := rows.Scan(&f.ID, &f.Brand, &f.Country, &f.Content, &f.SubNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return features, nil
}

// GetByBrand retrieves features for a specific brand
func (r *FeatureRepository) GetByBrand(ctx context.Context, brand string) ([]domain.Feature, error) {
	query := `
		SELECT id, brand, country, content, sub_number
		FROM feature
		WHERE brand = $1
		ORDER BY id, country, sub_number
	`

	rows, err := r.db.QueryContext(ctx, query, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to get features by brand: %w", err)
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		err := rows.Scan(&f.ID, &f.Brand, &f.Country, &f.Content, &f.SubNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return features, nil
}

// GetByProduct retrieves features for a specific product-country combination
func (r *FeatureRepository) GetByProduct(ctx context.Context, id int64, brand, country string) ([]domain.Feature, error) {
	query := `
		SELECT id, brand, country, content, sub_number
		FROM feature
		WHERE id = $1 AND brand = $2 AND country = $3
		ORDER BY sub_number
	`

	rows, err := r.db.QueryContext(ctx, query, id, brand, country)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}
	defer rows.Close()

	var features []domain.Feature
	for rows.Next() {
		var f domain.Feature
		err := rows.Scan(&f.ID, &f.Brand, &f.Country, &f.Content, &f.SubNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	return features, nil
}

// Count returns the total number of features
func (r *FeatureRepository) Count(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM feature"

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count features: %w", err)
	}

	return count, nil
}
