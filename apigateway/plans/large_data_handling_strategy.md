# Chiến Lược Xử Lý Data Cực Lớn - Phân Tích Nguy Hiểm & Giải Pháp

## ⚠️ Vấn Đề Với In-Memory Indexing Khi Data Cực Lớn

### Scenario: 1 triệu Products, 50 triệu Features, 50 triệu ProductInfos

```
Memory Estimation:
├─ Features (50M × 102 bytes):     ~5.1 GB
├─ ProductInfos (50M × 96 bytes):  ~4.8 GB
├─ Feature Index map[str]map[int64]map[str][]Feature: ~2 GB (pointers + overhead)
├─ ProductInfo Index map[str]map[int64]map[str][]ProductInfo: ~1.8 GB
└─ Total: ~14 GB minimum ❌ KHÔNG ỔN!
```

### Rủi Ro Cụ Thể

```
1. OOM (Out of Memory) Crash
   └─ Server crashed, tất cả request fail

2. Garbage Collection Pause
   └─ Stop-the-world GC kéo dài → P99 latency spike → API timeout

3. Slow Startup
   └─ Load 100M records vào memory = 30+ seconds

4. Deployment Problem
   └─ Container memory limit vượt quá → Pod bị terminate

5. Concurrent Request Problem
   └─ Multiple requests cùng load → Memory tăng exponential
```

---

## ✅ Giải Pháp Theo Tỷ Lệ Data

### Tier 1: Data Vừa Phải (< 100MB)
```
Use Case: Doanh nghiệp vừa, local deployment
Approach: In-Memory Indexing (Kế hoạch hiện tại) ✅
Pros:
  - Đơn giản, performance tốt nhất (O(1) lookup)
  - Không cần database query phức tạp
Cons:
  - Giới hạn data size
Memory: < 500 MB
Latency: < 100 ms
```

### Tier 2: Data Lớn (100 MB - 5 GB)
```
Use Case: Enterprise, cần scalability
Approach: Hybrid - Pagination + In-Memory Segment
Pros:
  - Scalable, memory controlled
  - Vẫn giữ performance tốt
Cons:
  - Phức tạp hơn, cần pagination logic
Memory: < 1 GB
Latency: 100-500 ms
```

### Tier 3: Data Cực Lớn (> 5 GB)
```
Use Case: Big Data, real-time processing
Approach: Stream Processing + Lazy Loading
Pros:
  - Scalable vô hạn
  - Memory constant
Cons:
  - Phức tạp, latency cao hơn
  - Cần caching strategy
Memory: < 500 MB (constant)
Latency: 500 ms - 2s
```

---

## 🎯 Giải Pháp Chi Tiết Cho Mỗi Tier

## Tier 1 Implementation (Hiện Tại - OK cho data vừa)

```go
// ✅ Ổn cho < 100MB
func (pm *ProductMerger) MergeAllProducts(products []domain.Product) []ProductDetailResponse {
    merger := NewProductMerger(features, productInfos)
    // Toàn bộ data trong memory
    return merger.MergeAllProducts(products)
}
```

---

## Tier 2 Implementation (Hybrid - Recommended cho 100MB - 5GB)

### Approach: Stream Processing + Batch Index Building

```go
package service

import (
    "context"
    "fmt"
    "sync"
)

// StreamMerger xử lý data lớn với pagination
type StreamMerger struct {
    featureRepo     *repository.FeatureRepository
    productInfoRepo *repository.ProductInfoRepository
    batchSize       int
    indexCache      *IndexCache
}

// IndexCache cache partial indexes để tránh rebuild
type IndexCache struct {
    mu sync.RWMutex
    cache map[string]interface{} // key = "brand|id", value = index
    ttl time.Duration
}

// NewStreamMerger tạo merger cho data lớn
func NewStreamMerger(fr *repository.FeatureRepository, 
    pr *repository.ProductInfoRepository, 
    batchSize int) *StreamMerger {
    return &StreamMerger{
        featureRepo:     fr,
        productInfoRepo: pr,
        batchSize:       batchSize,
        indexCache:      NewIndexCache(time.Hour),
    }
}

// MergeProductsPaginated xử lý products theo trang
func (sm *StreamMerger) MergeProductsPaginated(
    ctx context.Context,
    products []domain.Product,
    page, pageSize int,
) ([]ProductDetailResponse, error) {
    
    // 1. Paginate products
    start := (page - 1) * pageSize
    end := start + pageSize
    if end > len(products) {
        end = len(products)
    }
    paginatedProducts := products[start:end]
    
    // 2. Collect tất cả brands cần thiết từ trang này
    requiredBrands := collectBrands(paginatedProducts)
    
    // 3. Load features + productinfos ONLY cho brands này (Key optimization!)
    features, err := sm.featureRepo.GetByBrands(ctx, requiredBrands)
    if err != nil {
        return nil, err
    }
    
    productInfos, err := sm.productInfoRepo.GetByBrands(ctx, requiredBrands)
    if err != nil {
        productInfos = []domain.ProductInfo{}
    }
    
    // 4. Build temporary index (smaller scope)
    tempFeatureIndex := buildFeatureIndex(features)
    tempProductInfoIndex := buildProductInfoIndex(productInfos)
    
    // 5. Merge
    results := make([]ProductDetailResponse, 0, len(paginatedProducts))
    for _, product := range paginatedProducts {
        merged := mergeProductWithIndexes(product, tempFeatureIndex, tempProductInfoIndex)
        results = append(results, merged)
    }
    
    return results, nil
}

// MergeProductsStream process data secara stream, không load all
func (sm *StreamMerger) MergeProductsStream(
    ctx context.Context,
    productChan <-chan domain.Product,
    resultChan chan<- ProductDetailResponse,
) error {
    defer close(resultChan)
    
    // Process products in batches
    batch := make([]domain.Product, 0, sm.batchSize)
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case product, ok := <-productChan:
            if !ok {
                // Flush remaining batch
                if len(batch) > 0 {
                    sm.processBatch(ctx, batch, resultChan)
                }
                return nil
            }
            
            batch = append(batch, product)
            
            // Process when batch full
            if len(batch) >= sm.batchSize {
                if err := sm.processBatch(ctx, batch, resultChan); err != nil {
                    return err
                }
                batch = batch[:0] // Reset batch
            }
        }
    }
}

func (sm *StreamMerger) processBatch(
    ctx context.Context,
    batch []domain.Product,
    resultChan chan<- ProductDetailResponse,
) error {
    // Load features + productinfos for this batch only
    brands := collectBrands(batch)
    
    features, _ := sm.featureRepo.GetByBrands(ctx, brands)
    productInfos, _ := sm.productInfoRepo.GetByBrands(ctx, brands)
    
    // Merge
    tempFeatureIndex := buildFeatureIndex(features)
    tempProductInfoIndex := buildProductInfoIndex(productInfos)
    
    for _, product := range batch {
        merged := mergeProductWithIndexes(product, tempFeatureIndex, tempProductInfoIndex)
        select {
        case resultChan <- merged:
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return nil
}

// collectBrands extract unique brands từ products
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
```

### Handler untuk Tier 2 (Pagination)

```go
// GetAllProductsWithDetailsPaginated godoc
// @Summary Get products with pagination
// @Param page query int 1 "Page number"
// @Param page_size query int 20 "Page size"
func (h *ProductHandler) GetAllProductsWithDetailsPaginated(c echo.Context) error {
    ctx := c.Request().Context()
    
    page := 1
    pageSize := 20
    
    if p := c.QueryParam("page"); p != "" {
        fmt.Sscanf(p, "%d", &page)
    }
    if ps := c.QueryParam("page_size"); ps != "" {
        fmt.Sscanf(ps, "%d", &pageSize)
    }
    
    // Validate
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }
    
    // Get all products (only IDs + Brands, small overhead)
    products, err := h.productService.GetAllProducts(ctx)
    if err != nil {
        return c.JSON(500, "error")
    }
    
    // Merge paginated
    results, err := h.streamMerger.MergeProductsPaginated(ctx, products, page, pageSize)
    if err != nil {
        return c.JSON(500, "error")
    }
    
    return c.JSON(200, map[string]interface{}{
        "page": page,
        "size": pageSize,
        "data": results,
    })
}
```

---

## Tier 3 Implementation (Stream Processing - Cho data cực lớn)

### Approach: Database Joins + Caching

```go
package database

// ProductDetailQuery execute query tối ưu join trực tiếp ở DB
func (db *PostgresDB) QueryProductDetailsOptimized(
    ctx context.Context,
    brand string,
    limit, offset int,
) ([]ProductDetailDTO, error) {
    
    // Execute optimized SQL join
    query := `
    SELECT 
        p.id, p.brand,
        f.country, f.content, f.sub_number
    FROM products p
    LEFT JOIN features f ON p.id = f.id AND p.brand = f.brand
    WHERE p.brand = $1
    ORDER BY p.id, f.country, f.sub_number
    LIMIT $2 OFFSET $3
    `
    
    rows, err := db.QueryContext(ctx, query, brand, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    results := []ProductDetailDTO{}
    for rows.Next() {
        var detail ProductDetailDTO
        if err := rows.Scan(&detail.ID, &detail.Brand, 
            &detail.Country, &detail.Content, &detail.SubNumber); err != nil {
            return nil, err
        }
        
        // Merge ProductInfo from DataStore
        pi, _ := datastoreClient.GetProductInfo(ctx, detail.ID, detail.Brand, detail.Country)
        if pi != nil {
            detail.Place = pi.Place
            detail.Year = pi.Year
        }
        
        results = append(results, detail)
    }
    
    return results, rows.Err()
}

// GetProductDetailsStreamIterator return iterator, không load all
func (db *PostgresDB) GetProductDetailsStreamIterator(
    ctx context.Context,
    brand string,
) (*ProductDetailIterator, error) {
    
    query := `
    SELECT id, brand, country, content, sub_number
    FROM features
    WHERE brand = $1
    ORDER BY id, country, sub_number
    `
    
    rows, err := db.QueryContext(ctx, query, brand)
    if err != nil {
        return nil, err
    }
    
    return &ProductDetailIterator{
        rows:            rows,
        datastoreClient: datastoreClient,
    }, nil
}

type ProductDetailIterator struct {
    rows            *sql.Rows
    datastoreClient *DatastoreClient
}

func (it *ProductDetailIterator) Next(ctx context.Context) (*ProductDetailDTO, error) {
    if !it.rows.Next() {
        it.rows.Close()
        return nil, io.EOF
    }
    
    var detail ProductDetailDTO
    if err := it.rows.Scan(&detail.ID, &detail.Brand, 
        &detail.Country, &detail.Content, &detail.SubNumber); err != nil {
        return nil, err
    }
    
    // Lazy load ProductInfo from cache/DataStore
    pi, _ := it.datastoreClient.GetProductInfo(ctx, detail.ID, detail.Brand, detail.Country)
    if pi != nil {
        detail.Place = pi.Place
        detail.Year = pi.Year
    }
    
    return &detail, nil
}
```

---

## 📊 So Sánh Các Approach

| Aspect | Tier 1 (In-Memory) | Tier 2 (Pagination) | Tier 3 (Stream) |
|--------|-------------------|-------------------|-----------------|
| **Max Data** | < 100 MB | 100 MB - 5 GB | Unlimited |
| **Memory Usage** | 500 MB - 2 GB | < 1 GB | < 500 MB |
| **Query Latency** | < 100 ms | 100-500 ms | 500 ms - 2s |
| **Implementation** | Simple | Medium | Complex |
| **Scalability** | ❌ | ✅ | ✅✅ |
| **GC Impact** | High | Medium | Low |
| **Database Queries** | 1 | 2-10 | 1 + N (lazy) |
| **Caching** | Simple | TTL-based | Query-level |
| **Real-world Use** | MVP, Demo | Startup | Enterprise |

---

## 🚀 Recommendation Theo Tình Huống

### Situation 1: Startup (< 10K products)
```
✅ Use Tier 1 (In-Memory Indexing)
   - Đơn giản, nhanh
   - Không cần optimize sớm
```

### Situation 2: Growing Business (10K - 1M products)
```
✅ Use Tier 2 (Pagination + Stream)
   - Scalable
   - Predictable memory
   - User-friendly pagination
```

### Situation 3: Enterprise (> 1M products)
```
✅ Use Tier 3 (Stream Processing)
   - Xử lý unlimited data
   - Join tại database layer
   - Lazy load ProductInfo
   - Cache layer trước API
```

---

## 💡 Hybrid Strategy Tối Ưu (Recommended)

```go
package service

// AdaptiveMerger tự động chọn strategy dựa trên data size
type AdaptiveMerger struct {
    inMemoryMerger *ProductMerger    // Tier 1
    streamMerger   *StreamMerger     // Tier 2
    dbMerger       *DatabaseMerger   // Tier 3
}

func (am *AdaptiveMerger) Merge(ctx context.Context, products []domain.Product) ([]ProductDetailResponse, error) {
    totalDataSize := estimateDataSize(products)
    
    switch {
    case totalDataSize < 100*MB:
        // Tier 1: In-memory
        return am.inMemoryMerger.MergeAllProducts(products), nil
        
    case totalDataSize < 5*GB:
        // Tier 2: Stream with pagination
        return am.streamMerger.MergeProductsPaginated(ctx, products, 1, 100)
        
    default:
        // Tier 3: Database stream
        return am.dbMerger.StreamQueryProductDetails(ctx, "")
    }
}
```

---

## 🔒 Memory Safety Checks

```go
// Monitor memory before merge
func (h *ProductHandler) GetAllProductsWithDetails(c echo.Context) error {
    ctx := c.Request().Context()
    
    // Check available memory
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    availableMemory := getTotalSystemMemory() - m.Alloc
    
    products, _ := h.productService.GetAllProducts(ctx)
    estimatedSize := len(products) * 10 * MB // rough estimate
    
    if estimatedSize > availableMemory*0.8 {
        // Use Tier 3 (streaming)
        return h.handleLargeDataset(ctx, c)
    }
    
    // Safe to use Tier 1 or 2
    results, _ := h.productService.GetAllProductsWithDetails(ctx)
    return c.JSON(200, results)
}
```

---

## 📈 Performance Optimization Checklist

- [ ] Add database indexes on (brand, id, country)
- [ ] Implement query result caching (Redis)
- [ ] Use connection pooling for database
- [ ] Add pagination by default
- [ ] Monitor memory usage with Prometheus
- [ ] Set up auto-scaling based on memory
- [ ] Implement request-level timeout
- [ ] Use HTTP/2 push for streaming
- [ ] Add gzip compression
- [ ] Benchmark with production-like data

---

## ✅ Conclusion

```
Data Size              Best Approach                    Memory     Latency
─────────────────────────────────────────────────────────────────────────
< 100 MB              Tier 1 (In-Memory)              ✅ OK      ⚡ 100ms
100 MB - 5 GB         Tier 2 (Pagination/Stream)      ✅ OK      ⚡ 300ms
> 5 GB                Tier 3 (DB Streaming)           ✅ Safe    ⚡ 1s

⚠️ NEVER push data cực lớn vào memory!
✅ Use adaptive strategy + pagination + caching
```

