package service

import (
	"context"
	"fmt"

	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository"
)

// ProductService handles business logic for products
type ProductService struct {
	productRepo     *repository.ProductRepository
	featureRepo     *repository.FeatureRepository
	datastoreClient *database.DatastoreClient
}

// NewProductService creates a new ProductService instance
func NewProductService(
	productRepo *repository.ProductRepository,
	featureRepo *repository.FeatureRepository,
	dsClient *database.DatastoreClient,
) *ProductService {
	return &ProductService{
		productRepo:     productRepo,
		featureRepo:     featureRepo,
		datastoreClient: dsClient,
	}
}

// ==================== Product Operations ====================

// CreateProduct creates a new product
func (ps *ProductService) CreateProduct(ctx context.Context, product *domain.Product) error {
	if product.ID <= 0 {
		return fmt.Errorf("invalid product ID")
	}
	if product.Brand == "" {
		return fmt.Errorf("product brand cannot be empty")
	}

	return ps.productRepo.Create(ctx, product)
}

// GetProduct retrieves a product with all related data
func (ps *ProductService) GetProduct(ctx context.Context, id int64, brand string) (*ProductDTO, error) {
	product, err := ps.productRepo.GetByID(ctx, id, brand)
	if err != nil {
		return nil, err
	}

	// Get all ProductInfo for this product-brand from all countries
	productInfos, err := ps.datastoreClient.GetProductInfoByBrand(ctx, brand)
	if err != nil {
		// ProductInfo is optional, just log and continue
		productInfos = []domain.ProductInfo{}
	}

	// Get Features from SQL for this product
	features, err := ps.featureRepo.GetByBrand(ctx, brand)
	if err != nil {
		// Features are optional
		features = []domain.Feature{}
	}

	return &ProductDTO{
		Product:      product,
		ProductInfos: productInfos,
		Features:     features,
	}, nil
}

// GetProductsByBrand retrieves all products for a brand
func (ps *ProductService) GetProductsByBrand(ctx context.Context, brand string) ([]domain.Product, error) {
	return ps.productRepo.GetByBrand(ctx, brand)
}

// GetAllProducts retrieves all products
func (ps *ProductService) GetAllProducts(ctx context.Context) ([]domain.Product, error) {
	return ps.productRepo.GetAll(ctx)
}

// DeleteProduct deletes a product
func (ps *ProductService) DeleteProduct(ctx context.Context, id int64, brand string) error {
	return ps.productRepo.Delete(ctx, id, brand)
}

// UpdateProductRevision updates the product revision
func (ps *ProductService) UpdateProductRevision(ctx context.Context, id int64, brand string) error {
	return ps.productRepo.UpdateRevision(ctx, id, brand)
}

// ==================== Feature Operations (SQL-based) ====================

// GetFeaturesByProduct retrieves all features for a product-country combination
func (ps *ProductService) GetFeaturesByProduct(ctx context.Context, id int64, brand, country string) ([]domain.Feature, error) {
	return ps.featureRepo.GetByProduct(ctx, id, brand, country)
}

// GetFeaturesByBrand retrieves all features for a brand
func (ps *ProductService) GetFeaturesByBrand(ctx context.Context, brand string) ([]domain.Feature, error) {
	return ps.featureRepo.GetByBrand(ctx, brand)
}

// ==================== ProductInfo Operations ====================

// CreateProductInfo creates ProductInfo for a product in a country
func (ps *ProductService) CreateProductInfo(ctx context.Context, productInfo *domain.ProductInfo) error {
	if productInfo.ID <= 0 {
		return fmt.Errorf("invalid product ID")
	}
	if productInfo.Brand == "" {
		return fmt.Errorf("brand cannot be empty")
	}
	if productInfo.Country == "" {
		return fmt.Errorf("country cannot be empty")
	}

	return ps.datastoreClient.SaveProductInfo(ctx, productInfo)
}

// GetProductInfo retrieves ProductInfo for a specific product-country combination
func (ps *ProductService) GetProductInfo(ctx context.Context, id int64, brand, country string) (*domain.ProductInfo, error) {
	return ps.datastoreClient.GetProductInfo(ctx, id, brand, country)
}

// GetProductInfosByBrand retrieves all ProductInfo for a brand
func (ps *ProductService) GetProductInfosByBrand(ctx context.Context, brand string) ([]domain.ProductInfo, error) {
	return ps.datastoreClient.GetProductInfoByBrand(ctx, brand)
}

// ==================== Data Transfer Objects ====================

// ProductDTO represents a complete product with all related data
type ProductDTO struct {
	Product      *domain.Product      `json:"product"`
	ProductInfos []domain.ProductInfo `json:"product_infos"`
	Features     []domain.Feature     `json:"features"`
}

// GetStatistics retrieves data statistics
func (ps *ProductService) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	totalProducts, err := ps.productRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	totalFeatures, err := ps.featureRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_products": totalProducts,
		"total_features": totalFeatures,
	}, nil
}
