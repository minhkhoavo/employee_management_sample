package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/locvowork/employee_management_sample/apigateway/internal/database"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository"
)

// ProductMerger handles merging product data from SQL and DataStore
type ProductMerger struct {
	ProductRepo     *repository.ProductRepository
	featureRepo     *repository.FeatureRepository
	productInfoRepo *database.DatastoreClient
	batchSize       int
	numWorkers      int
}

// NewProductMerger creates a new merger
func NewProductMerger(
	pr *repository.ProductRepository,
	fr *repository.FeatureRepository,
	pir *database.DatastoreClient,
	batchSize, numWorkers int,
) *ProductMerger {
	return &ProductMerger{
		ProductRepo:     pr,
		featureRepo:     fr,
		productInfoRepo: pir,
		batchSize:       batchSize,
		numWorkers:      numWorkers,
	}
}

// ============================================================================
// Phase 1: Merge In-Memory (Sequential)
// ============================================================================

// MergeProductBatch merges a single batch of products in memory
// This function does ALL the work: fetch data + merge locally
func (pm *ProductMerger) MergeProductBatch(
	ctx context.Context,
	batch []domain.Product,
) ([]domain.ProductDetailResponse, error) {

	// 1. Collect unique brands from batch
	brands := collectBrands(batch)

	// 2. Fetch features ONLY for brands in this batch
	features, err := pm.featureRepo.GetByBrands(ctx, brands)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}

	// 3. Fetch ProductInfos ONLY for brands in this batch
	var productInfos []domain.ProductInfo
	for _, brand := range brands {
		infos, _ := pm.productInfoRepo.GetProductInfoByBrand(ctx, brand)
		productInfos = append(productInfos, infos...)
	}

	// 4. Build IN-MEMORY indexes for this batch
	featureIndex := buildFeatureIndexLocal(features)
	productInfoIndex := buildProductInfoIndexLocal(productInfos)

	// 5. Merge products locally
	results := make([]domain.ProductDetailResponse, 0, len(batch))
	for _, product := range batch {
		merged := mergeProductLocal(product, featureIndex, productInfoIndex)
		results = append(results, merged)
	}

	return results, nil
}

// buildFeatureIndexLocal creates index: [Brand][ID][Country] -> []Feature
func buildFeatureIndexLocal(features []domain.Feature) map[string]map[int64]map[string][]domain.Feature {
	index := make(map[string]map[int64]map[string][]domain.Feature)

	for _, f := range features {
		if index[f.Brand] == nil {
			index[f.Brand] = make(map[int64]map[string][]domain.Feature)
		}
		if index[f.Brand][f.ID] == nil {
			index[f.Brand][f.ID] = make(map[string][]domain.Feature)
		}
		index[f.Brand][f.ID][f.Country] = append(
			index[f.Brand][f.ID][f.Country], f)
	}

	return index
}

// buildProductInfoIndexLocal creates index: [Brand][ID][Country] -> []ProductInfo
func buildProductInfoIndexLocal(infos []domain.ProductInfo) map[string]map[int64]map[string][]domain.ProductInfo {
	index := make(map[string]map[int64]map[string][]domain.ProductInfo)

	for _, pi := range infos {
		if index[pi.Brand] == nil {
			index[pi.Brand] = make(map[int64]map[string][]domain.ProductInfo)
		}
		if index[pi.Brand][pi.ID] == nil {
			index[pi.Brand][pi.ID] = make(map[string][]domain.ProductInfo)
		}
		index[pi.Brand][pi.ID][pi.Country] = append(
			index[pi.Brand][pi.ID][pi.Country], pi)
	}

	return index
}

// mergeProductLocal merges a single product with features and product infos
func mergeProductLocal(
	product domain.Product,
	featureIndex map[string]map[int64]map[string][]domain.Feature,
	productInfoIndex map[string]map[int64]map[string][]domain.ProductInfo,
) domain.ProductDetailResponse {

	resp := domain.ProductDetailResponse{
		Item: domain.ProductItemDTO{
			ID:    product.ID,
			Brand: product.Brand,
		},
		Details: []domain.ProductDetailDTO{},
	}

	// Get unique countries
	countrySet := make(map[string]bool)

	// From features
	if featureIndex[product.Brand] != nil && featureIndex[product.Brand][product.ID] != nil {
		for country := range featureIndex[product.Brand][product.ID] {
			countrySet[country] = true
		}
	}

	// From product infos
	if productInfoIndex[product.Brand] != nil && productInfoIndex[product.Brand][product.ID] != nil {
		for country := range productInfoIndex[product.Brand][product.ID] {
			countrySet[country] = true
		}
	}

	// Merge for each country
	for country := range countrySet {
		feats := featureIndex[product.Brand][product.ID][country]
		infos := productInfoIndex[product.Brand][product.ID][country]

		merged := mergeBySubNumber(product, feats, infos, country)
		resp.Details = append(resp.Details, merged...)
	}

	return resp
}

// mergeBySubNumber merges features and product infos by sub number
func mergeBySubNumber(
	product domain.Product,
	features []domain.Feature,
	productInfos []domain.ProductInfo,
	country string,
) []domain.ProductDetailDTO {

	result := []domain.ProductDetailDTO{}

	// Map ProductInfo by SubNumber for O(1) lookup
	piMap := make(map[int]domain.ProductInfo)
	for _, pi := range productInfos {
		piMap[pi.SubNumber] = pi
	}

	// Merge features with productinfo
	featureSubNumbers := make(map[int]bool)
	for _, f := range features {
		detail := domain.ProductDetailDTO{
			ID:        f.ID,
			Brand:     f.Brand,
			Country:   country,
			SubNumber: f.SubNumber,
			Content:   f.Content,
		}

		// Add ProductInfo if exists
		if pi, ok := piMap[f.SubNumber]; ok {
			detail.Place = pi.Place
			detail.Year = pi.Year
		}

		result = append(result, detail)
		featureSubNumbers[f.SubNumber] = true
	}

	// Add ProductInfos without matching Feature
	for _, pi := range productInfos {
		if !featureSubNumbers[pi.SubNumber] {
			detail := domain.ProductDetailDTO{
				ID:        pi.ID,
				Brand:     pi.Brand,
				Country:   country,
				Place:     pi.Place,
				Year:      pi.Year,
				SubNumber: pi.SubNumber,
			}
			result = append(result, detail)
		}
	}

	return result
}

// ============================================================================
// Phase 2: Concurrent Batch Processing (Fan-In/Fan-Out)
// ============================================================================

// BatchedProductResult represents a merged batch result
type BatchedProductResult struct {
	BatchIdx int // For ordering
	Results  []domain.ProductDetailResponse
	Error    error
}

// MergeProductsConcurrent processes products concurrently using fan-in/fan-out
func (pm *ProductMerger) MergeProductsConcurrent(
	ctx context.Context,
) ([]domain.ProductDetailResponse, error) {

	// 1. Fetch all products
	products, err := pm.ProductRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	if len(products) == 0 {
		return []domain.ProductDetailResponse{}, nil
	}

	// 2. Split into batches
	batches := splitIntoBatches(products, pm.batchSize)
	fmt.Printf("[CONCURRENT] Total products: %d, Batch size: %d, Number of batches: %d\n", len(products), pm.batchSize, len(batches))

	// 3. Fan-Out: Send batches to workers
	batchChan := make(chan *BatchWork, len(batches))
	for idx, batch := range batches {
		batchChan <- &BatchWork{
			BatchIdx: idx,
			Products: batch,
		}
	}
	close(batchChan)

	// 4. Fan-In: Process results from workers
	resultChan := make(chan *BatchedProductResult, len(batches))
	var wg sync.WaitGroup

	// Spawn workers
	numWorkers := pm.numWorkers
	if numWorkers > len(batches) {
		numWorkers = len(batches)
	}
	fmt.Printf("[CONCURRENT] Spawning %d workers to process %d batches\n", numWorkers, len(batches))

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pm.worker(ctx, batchChan, resultChan, &wg)
	}

	// Close resultChan when all workers done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 5. Collect all results
	results := make(map[int][]domain.ProductDetailResponse)
	totalProcessed := 0

	for batchResult := range resultChan {
		if batchResult.Error != nil {
			return nil, fmt.Errorf("batch %d failed: %w", batchResult.BatchIdx, batchResult.Error)
		}

		results[batchResult.BatchIdx] = batchResult.Results
		totalProcessed++
		fmt.Printf("[CONCURRENT] Batch %d completed (%d/%d) - %d products\n", batchResult.BatchIdx, totalProcessed, len(batches), len(batchResult.Results))

		results[batchResult.BatchIdx] = batchResult.Results
		totalProcessed++
	}

	// 6. Merge results in order
	finalResults := make([]domain.ProductDetailResponse, 0, len(products))
	for i := 0; i < len(batches); i++ {
		finalResults = append(finalResults, results[i]...)
	}

	return finalResults, nil
}

// BatchWork represents work to be done on a batch
type BatchWork struct {
	BatchIdx int
	Products []domain.Product
}

// worker processes batches from the work channel
func (pm *ProductMerger) worker(
	ctx context.Context,
	batchChan <-chan *BatchWork,
	resultChan chan<- *BatchedProductResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case batch, ok := <-batchChan:
			if !ok {
				// No more work
				return
			}

			fmt.Printf("[WORKER] Processing batch %d with %d products\n", batch.BatchIdx, len(batch.Products))

			// Process batch
			results, err := pm.MergeProductBatch(ctx, batch.Products)
			resultChan <- &BatchedProductResult{
				BatchIdx: batch.BatchIdx,
				Results:  results,
				Error:    err,
			}
		}
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// collectBrands extracts unique brands from products
func collectBrands(products []domain.Product) []string {
	brandSet := make(map[string]bool)
	for _, p := range products {
		brandSet[p.Brand] = true
	}

	brands := make([]string, 0, len(brandSet))
	for b := range brandSet {
		brands = append(brands, b)
	}
	return brands
}

// splitIntoBatches splits products into batches
func splitIntoBatches(products []domain.Product, batchSize int) [][]domain.Product {
	if batchSize <= 0 {
		batchSize = 100
	}

	batches := make([][]domain.Product, 0)
	for i := 0; i < len(products); i += batchSize {
		end := i + batchSize
		if end > len(products) {
			end = len(products)
		}
		batches = append(batches, products[i:end])
	}

	return batches
}
