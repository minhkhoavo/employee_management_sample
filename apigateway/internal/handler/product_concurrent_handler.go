package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
)

// ProductMergeHandler handles product merge requests
type ProductMergeHandler struct {
	merger *service.ProductMerger
}

// NewProductMergeHandler creates a new handler
func NewProductMergeHandler(merger *service.ProductMerger) *ProductMergeHandler {
	return &ProductMergeHandler{
		merger: merger,
	}
}

// GetAllProductsWithDetailsMerged godoc
// @Summary Get all products with merged details (sequential)
// @Description Lấy tất cả products merged từ SQL + DataStore (in-memory indexing)
// @Tags Products
// @Accept json
// @Produce json
// @Success 200 {array} domain.ProductDetailResponse
// @Router /products/details-merged [get]
func (h *ProductMergeHandler) GetAllProductsWithDetailsMerged(c echo.Context) error {
	ctx := c.Request().Context()
	start := time.Now()

	// Get all products
	products, err := h.merger.ProductRepo.GetAll(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if len(products) == 0 {
		duration := time.Since(start)
		fmt.Printf("[SEQUENTIAL] No products found - Time: %v\n", duration)
		return c.JSON(http.StatusOK, []interface{}{})
	}

	// Merge using in-memory indexing (single batch)
	results, err := h.merger.MergeProductBatch(ctx, products)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	duration := time.Since(start)
	fmt.Printf("[SEQUENTIAL] Merged %d products - Time: %v\n", len(products), duration)

	return c.JSON(http.StatusOK, results)
}

// GetAllProductsWithDetailsConcurrent godoc
// @Summary Get all products with merged details (concurrent)
// @Description Lấy tất cả products merged concurrently (fan-in/fan-out)
// @Tags Products
// @Accept json
// @Produce json
// @Success 200 {array} domain.ProductDetailResponse
// @Router /products/details-concurrent [get]
func (h *ProductMergeHandler) GetAllProductsWithDetailsConcurrent(c echo.Context) error {
	ctx := c.Request().Context()
	start := time.Now()

	results, err := h.merger.MergeProductsConcurrent(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	duration := time.Since(start)
	fmt.Printf("[CONCURRENT] Merged %d products - Time: %v\n", len(results), duration)

	return c.JSON(http.StatusOK, results)
}
