package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
)

// ProductRepository handles all database operations for Product
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new instance of ProductRepository
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create inserts a new product into the database
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	query := `
		INSERT INTO product (id, brand, revision)
		VALUES ($1, $2, $3)
		ON CONFLICT (brand, id) DO UPDATE SET revision = $3
	`

	_, err := r.db.ExecContext(ctx, query, product.ID, product.Brand, product.Revision)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

// GetByID retrieves a product by ID and Brand
func (r *ProductRepository) GetByID(ctx context.Context, id int64, brand string) (*domain.Product, error) {
	query := `
		SELECT id, brand, revision
		FROM product
		WHERE id = $1 AND brand = $2
	`

	var product domain.Product
	err := r.db.QueryRowContext(ctx, query, id, brand).Scan(&product.ID, &product.Brand, &product.Revision)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

// GetByBrand retrieves all products by brand
func (r *ProductRepository) GetByBrand(ctx context.Context, brand string) ([]domain.Product, error) {
	query := `
		SELECT id, brand, revision
		FROM product
		WHERE brand = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, brand)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(&product.ID, &product.Brand, &product.Revision); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return products, nil
}

// GetAll retrieves all products
func (r *ProductRepository) GetAll(ctx context.Context) ([]domain.Product, error) {
	query := `
		SELECT id, brand, revision
		FROM product
		ORDER BY brand, id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(&product.ID, &product.Brand, &product.Revision); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return products, nil
}

// UpdateRevision increments the revision for a product
func (r *ProductRepository) UpdateRevision(ctx context.Context, id int64, brand string) error {
	query := `
		UPDATE product
		SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND brand = $2
	`

	result, err := r.db.ExecContext(ctx, query, id, brand)
	if err != nil {
		return fmt.Errorf("failed to update revision: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}

// Delete removes a product
func (r *ProductRepository) Delete(ctx context.Context, id int64, brand string) error {
	query := `
		DELETE FROM product
		WHERE id = $1 AND brand = $2
	`

	_, err := r.db.ExecContext(ctx, query, id, brand)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	return nil
}

// Batch operations

// BatchCreate inserts multiple products
func (r *ProductRepository) BatchCreate(ctx context.Context, products []domain.Product) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, product := range products {
		query := `
			INSERT INTO product (id, brand, revision)
			VALUES ($1, $2, $3)
			ON CONFLICT (brand, id) DO UPDATE SET revision = $3
		`

		_, err := tx.ExecContext(ctx, query, product.ID, product.Brand, product.Revision)
		if err != nil {
			return fmt.Errorf("failed to insert product: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Count returns total number of products
func (r *ProductRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM product`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	return count, nil
}
