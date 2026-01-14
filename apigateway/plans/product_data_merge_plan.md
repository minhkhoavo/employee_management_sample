# Kế Hoạch Merge Dữ Liệu Product, Feature, ProductInfo

## 📋 Tổng Quan

Dự án hiện tại có 3 nguồn dữ liệu khác nhau:
- **SQL Database (PostgreSQL)**: Lưu Product, Feature
- **GCP Datastore**: Lưu ProductInfo
- **Yêu cầu**: Merge thành một response duy nhất với đầy đủ thông tin

---

## 🏗️ Kiến Trúc Dữ Liệu

### Hiện Tại
```
SQL Database:
├── Product (ID, Brand, Revision)
└── Feature (ID, Brand, Country, Content, SubNumber)

GCP Datastore:
└── ProductInfo (ID, Brand, Country, Place, Year, SubNumber)
```

### Mục Tiêu Response
```go
type ProductDetailResponse struct {
    Item    ProductItemDTO     `json:"item"`      // Thông tin Product cơ bản
    Details []ProductDetailDTO `json:"details"`   // Thông tin chi tiết merged
}

type ProductItemDTO struct {
    ID    int64  `json:"id"`
    Brand string `json:"brand"`
}

type ProductDetailDTO struct {
    ID        int64  `json:"id"`
    Brand     string `json:"brand"`
    Country   string `json:"country"`
    Place     string `json:"place"`        // Từ ProductInfo
    Year      int    `json:"year"`         // Từ ProductInfo
    SubNumber int    `json:"sub_number"`   // Từ Feature & ProductInfo
    Content   string `json:"content"`      // Từ Feature
}
```

---

## 🔄 Quy Trình Merge Dữ Liệu

### Phase 1: Lấy Dữ Liệu từ SQL
```
1. Gọi ProductRepository.GetAll() 
   └─ Lấy danh sách tất cả Products (ID, Brand, Revision)

2. Gọi FeatureRepository.GetAll()
   └─ Lấy danh sách tất cả Features (ID, Brand, Country, Content, SubNumber)

3. Tạo in-memory index:
   features_map[Brand][ID][Country] = []Feature
   └─ Giúp lookup O(1) thay vì O(n)
```

### Phase 2: Lấy Dữ Liệu từ DataStore
```
1. Gọi DatastoreClient.GetAllProductInfos()
   └─ Lấy danh sách tất cả ProductInfos

2. Tạo in-memory index:
   productinfo_map[Brand][ID][Country] = []ProductInfo
   └─ Giúp lookup O(1)
```

### Phase 3: Merge Data
```
Cho mỗi Product:
  ├─ Tạo ProductItemDTO từ Product cơ bản
  ├─ Lấy tất cả Countries từ Features
  ├─ Cho mỗi Country:
  │  ├─ Lấy Features của (ID, Brand, Country)
  │  ├─ Lấy ProductInfos của (ID, Brand, Country)
  │  ├─ Merge Feature + ProductInfo (by SubNumber)
  │  └─ Append vào Details
  └─ Trả về ProductDetailResponse
```

---

## ⚡ Tối Ưu Hiệu Năng (Performance Tips)

### 1. **In-Memory Indexing** (Rất Quan Trọng)
```
❌ KHÔ HIỆU QUẢ - O(n) lookup mỗi lần:
for product := range products {
    for feature := range features {  // n*m complexity
        if feature.ID == product.ID && feature.Brand == product.Brand {
            ...
        }
    }
}

✅ TỐI ƯU - O(1) lookup:
// Build index once
featureIndex := BuildFeatureIndex(features)  // O(n)
for product := range products {
    details := featureIndex[product.Brand][product.ID]  // O(1)
}
```

### 2. **Batch Queries từ DataStore**
```
❌ KHÔ HIỆU QUẢ - N queries:
for product := range products {
    productinfos := datastore.GetByBrand(product.Brand)  // N requests
}

✅ TỐI ƯU - 1 query duy nhất:
allProductInfos := datastore.GetAll()  // 1 request
productInfoIndex := BuildIndex(allProductInfos)
```

### 3. **Lazy Loading (Nếu cần)**
```
Nếu dataset quá lớn:
- Pagination: Limit products ở mỗi request
- Caching: Cache ProductInfo từ DataStore (TTL: 1 hour)
- Concurrency: Fetch SQL + DataStore đồng thời
```

### 4. **Memory Considerations**
```
Estimate memory usage:
- 1000 Products × (ID:8 + Brand:20 + Revision:8) = ~36KB
- 10000 Features × (ID:8 + Brand:20 + Country:20 + Content:50 + SubNum:4) = ~920KB
- 10000 ProductInfos × (ID:8 + Brand:20 + Country:20 + Place:20 + Year:4 + SubNum:4) = ~760KB
Total: ~2MB (acceptable)
```

---

## 🛠️ Implementation Steps (Chi Tiết)

### Step 1: Định Nghĩa DTOs
**File**: `internal/handler/dto.go`

```go
package handler

// ProductDetailResponse là response chính
type ProductDetailResponse struct {
    Item    ProductItemDTO     `json:"item"`
    Details []ProductDetailDTO `json:"details"`
}

// ProductItemDTO chứa info cơ bản
type ProductItemDTO struct {
    ID    int64  `json:"id"`
    Brand string `json:"brand"`
}

// ProductDetailDTO chứa merged info
type ProductDetailDTO struct {
    ID        int64  `json:"id"`
    Brand     string `json:"brand"`
    Country   string `json:"country"`
    Place     string `json:"place"`
    Year      int    `json:"year"`
    SubNumber int    `json:"sub_number"`
    Content   string `json:"content"`
}
```

### Step 2: Tạo Index Builders
**File**: `internal/service/product_merger.go`

```go
package service

import (
    "github.com/locvowork/employee_management_sample/apigateway/internal/domain"
    "github.com/locvowork/employee_management_sample/apigateway/internal/handler"
)

// ProductMerger chịu trách nhiệm merge dữ liệu
type ProductMerger struct {
    features     []domain.Feature
    productInfos []domain.ProductInfo
    featureIndex map[string]map[int64]map[string][]domain.Feature
    productInfoIndex map[string]map[int64]map[string][]domain.ProductInfo
}

// NewProductMerger tạo merger mới
func NewProductMerger(features []domain.Feature, productInfos []domain.ProductInfo) *ProductMerger {
    return &ProductMerger{
        features:     features,
        productInfos: productInfos,
        featureIndex: buildFeatureIndex(features),
        productInfoIndex: buildProductInfoIndex(productInfos),
    }
}

// buildFeatureIndex tạo index: [Brand][ID][Country] -> []Feature
func buildFeatureIndex(features []domain.Feature) map[string]map[int64]map[string][]domain.Feature {
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

// buildProductInfoIndex tạo index: [Brand][ID][Country] -> []ProductInfo
func buildProductInfoIndex(infos []domain.ProductInfo) map[string]map[int64]map[string][]domain.ProductInfo {
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
```

### Step 3: Implement Merge Logic
**File**: `internal/service/product_merger.go` (tiếp)

```go
// MergeProduct merge một product với features + productinfos
func (pm *ProductMerger) MergeProduct(product domain.Product) *handler.ProductDetailResponse {
    resp := &handler.ProductDetailResponse{
        Item: handler.ProductItemDTO{
            ID:    product.ID,
            Brand: product.Brand,
        },
        Details: []handler.ProductDetailDTO{},
    }
    
    // Lấy countries duy nhất
    countries := pm.getCountriesForProduct(product.Brand, product.ID)
    
    for _, country := range countries {
        // Lấy features cho country này
        features := pm.getFeatures(product.Brand, product.ID, country)
        
        // Lấy product infos cho country này
        productInfos := pm.getProductInfos(product.Brand, product.ID, country)
        
        // Merge bằng SubNumber
        details := pm.mergeBySubNumber(features, productInfos, country)
        resp.Details = append(resp.Details, details...)
    }
    
    return resp
}

// getCountriesForProduct lấy danh sách countries unique
func (pm *ProductMerger) getCountriesForProduct(brand string, id int64) []string {
    countrySet := make(map[string]bool)
    
    // Từ features
    if pm.featureIndex[brand] != nil && pm.featureIndex[brand][id] != nil {
        for country := range pm.featureIndex[brand][id] {
            countrySet[country] = true
        }
    }
    
    // Từ product infos
    if pm.productInfoIndex[brand] != nil && pm.productInfoIndex[brand][id] != nil {
        for country := range pm.productInfoIndex[brand][id] {
            countrySet[country] = true
        }
    }
    
    countries := make([]string, 0, len(countrySet))
    for c := range countrySet {
        countries = append(countries, c)
    }
    return countries
}

// getFeatures lấy features
func (pm *ProductMerger) getFeatures(brand string, id int64, country string) []domain.Feature {
    if pm.featureIndex[brand] == nil {
        return []domain.Feature{}
    }
    if pm.featureIndex[brand][id] == nil {
        return []domain.Feature{}
    }
    return pm.featureIndex[brand][id][country]
}

// getProductInfos lấy product infos
func (pm *ProductMerger) getProductInfos(brand string, id int64, country string) []domain.ProductInfo {
    if pm.productInfoIndex[brand] == nil {
        return []domain.ProductInfo{}
    }
    if pm.productInfoIndex[brand][id] == nil {
        return []domain.ProductInfo{}
    }
    return pm.productInfoIndex[brand][id][country]
}

// mergeBySubNumber merge feature + productinfo by SubNumber
func (pm *ProductMerger) mergeBySubNumber(
    features []domain.Feature,
    productInfos []domain.ProductInfo,
    country string,
) []handler.ProductDetailDTO {
    
    result := []handler.ProductDetailDTO{}
    
    // Map ProductInfo by SubNumber for O(1) lookup
    piMap := make(map[int]domain.ProductInfo)
    for _, pi := range productInfos {
        piMap[pi.SubNumber] = pi
    }
    
    // Merge features dengan productinfo
    for _, f := range features {
        detail := handler.ProductDetailDTO{
            ID:        f.ID,
            Brand:     f.Brand,
            Country:   country,
            SubNumber: f.SubNumber,
            Content:   f.Content,
        }
        
        // Tambm thêm ProductInfo nếu có
        if pi, ok := piMap[f.SubNumber]; ok {
            detail.Place = pi.Place
            detail.Year = pi.Year
        }
        
        result = append(result, detail)
    }
    
    // Thêm ProductInfos không có Feature tương ứng
    featureSubNumbers := make(map[int]bool)
    for _, f := range features {
        featureSubNumbers[f.SubNumber] = true
    }
    
    for _, pi := range productInfos {
        if !featureSubNumbers[pi.SubNumber] {
            detail := handler.ProductDetailDTO{
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

// MergeAllProducts merge tất cả products
func (pm *ProductMerger) MergeAllProducts(products []domain.Product) []handler.ProductDetailResponse {
    results := make([]handler.ProductDetailResponse, 0, len(products))
    
    for _, product := range products {
        merged := pm.MergeProduct(product)
        results = append(results, *merged)
    }
    
    return results
}
```

### Step 4: Integrate với Service/Handler
**File**: `internal/service/product_service.go` (thêm method)

```go
// GetAllProductsWithDetails lấy tất cả products + merge data
func (ps *ProductService) GetAllProductsWithDetails(ctx context.Context) ([]handler.ProductDetailResponse, error) {
    // 1. Fetch all data
    products, err := ps.productRepo.GetAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch products: %w", err)
    }
    
    features, err := ps.featureRepo.GetAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch features: %w", err)
    }
    
    productInfos, err := ps.datastoreClient.GetAllProductInfos(ctx)
    if err != nil {
        // ProductInfo optional, continue with empty list
        productInfos = []domain.ProductInfo{}
    }
    
    // 2. Merge data
    merger := NewProductMerger(features, productInfos)
    results := merger.MergeAllProducts(products)
    
    return results, nil
}
```

### Step 5: Expose qua Handler
**File**: `internal/handler/product_handler.go` (thêm endpoint)

```go
// GetAllProductsWithDetails godoc
// @Summary Get all products with merged details
// @Description Lấy tất cả products merged từ SQL + DataStore
// @Tags Products
// @Accept json
// @Produce json
// @Success 200 {array} handler.ProductDetailResponse
// @Router /products/details [get]
func (h *ProductHandler) GetAllProductsWithDetails(c echo.Context) error {
    ctx := c.Request().Context()
    
    results, err := h.productService.GetAllProductsWithDetails(ctx)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": err.Error(),
        })
    }
    
    return c.JSON(http.StatusOK, results)
}
```

---

## 📊 Complexity Analysis

### Time Complexity
```
Build Indexes:     O(n + m)
Merge Single:      O(k) k = countries per product
Merge All:         O(p * k + n + m)
                   where p = products, n = features, m = productinfos

Overall:           O(p * k + n + m) ≈ O(n) for small k
```

### Space Complexity
```
Feature Index:     O(n)
ProductInfo Index: O(m)
Results:           O(n + m)

Overall:           O(n + m)
```

---

## ⚠️ Error Handling & Edge Cases

### 1. Product không có Features hoặc ProductInfo
```go
✅ Được phép, Details sẽ là empty array
❌ Không được phép vì yêu cầu phải có details
```

### 2. Mismatch SubNumber giữa Feature và ProductInfo
```go
✅ Giải pháp: Merge independently, cho phép left/right join
```

### 3. DataStore không khả dụng
```go
✅ Graceful degradation: Trả về features only, ProductInfo fields = null
```

### 4. Dataset quá lớn
```go
✅ Giải pháp: Implement pagination trong Step 3
```

---

## 🔍 Testing Strategy

### Unit Tests
```go
1. Test buildFeatureIndex
2. Test buildProductInfoIndex
3. Test mergeBySubNumber
4. Test getCountriesForProduct
5. Test edge cases (empty list, nil values)
```

### Integration Tests
```go
1. Test GetAllProductsWithDetails end-to-end
2. Test with sample data (small, medium, large)
3. Benchmark performance
```

---

## 📈 Monitoring & Profiling

### Metrics to Track
```
- Query latency (SQL + DataStore)
- Merge time
- Memory usage
- Cache hit rate (nếu implement cache)
- Response size
```

### Profiling Commands
```bash
# CPU Profile
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Memory Profile
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

---

## 🎯 Summary

| Aspect | Solution |
|--------|----------|
| **Data Sources** | SQL (Product, Feature) + DataStore (ProductInfo) |
| **Merge Strategy** | Build in-memory indexes, merge by Brand/ID/Country/SubNumber |
| **Performance** | O(n) with 3-tier indexing |
| **Error Handling** | Graceful degradation, optional ProductInfo |
| **Scalability** | Pagination ready, batch queries |
| **Maintainability** | Separate merger service, clean DTOs |

