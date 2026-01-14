package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
)

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	productService *service.ProductService
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// ==================== Response Types ====================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ==================== Product Endpoints ====================

// GetAllProducts handles GET /api/products
func (ph *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger.InfoLog(ctx, "GET /api/products")

	products, err := ph.productService.GetAllProducts(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    products,
	})
}

// GetProductByID handles GET /api/products/:id
func (ph *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get("id")
	brand := r.URL.Query().Get("brand")

	if idStr == "" || brand == "" {
		respondError(w, http.StatusBadRequest, "id and brand parameters required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	logger.InfoLog(ctx, "GET /api/products - id=%d, brand=%s", id, brand)

	product, err := ph.productService.GetProduct(ctx, id, brand)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    product,
	})
}

// GetProductsByBrand handles GET /api/products/brand/:brand
func (ph *ProductHandler) GetProductsByBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	brand := r.URL.Query().Get("brand")
	if brand == "" {
		respondError(w, http.StatusBadRequest, "brand parameter required")
		return
	}

	logger.InfoLog(ctx, "GET /api/products/brand - brand=%s", brand)

	products, err := ph.productService.GetProductsByBrand(ctx, brand)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    products,
	})
}

// ==================== Feature Endpoints ====================

// GetFeatures handles GET /api/features
func (ph *ProductHandler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get("id")
	brand := r.URL.Query().Get("brand")
	country := r.URL.Query().Get("country")

	if idStr == "" || brand == "" || country == "" {
		respondError(w, http.StatusBadRequest, "id, brand, and country parameters required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	logger.InfoLog(ctx, "GET /api/features - id=%d, brand=%s, country=%s", id, brand, country)

	features, err := ph.productService.GetFeaturesByProduct(ctx, id, brand, country)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    features,
	})
}

// GetStatistics handles GET /api/statistics
func (ph *ProductHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger.InfoLog(ctx, "GET /api/statistics")

	stats, err := ph.productService.GetStatistics(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// ==================== Helper Functions ====================

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, APIResponse{
		Success: false,
		Error:   message,
	})
}
