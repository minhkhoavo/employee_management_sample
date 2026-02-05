# Excel Large Data Export API

## Overview

API để download file Excel chứa 200,000+ sản phẩm sử dụng **excelize v2** với **streaming mode** để đảm bảo:
- Không vượt quá **1.5GB RAM** 
- Logging chi tiết thời gian và memory usage
- Compatible với **Go 1.17+**

## Endpoints

### 1. Export Products to Excel
```
GET /api/v1/excel/products/export
```

**Query Parameters:**
| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `count` | int | 200000 | 500000 | Số lượng products |
| `batch_size` | int | 10000 | - | Batch size cho logging |
| `max_memory_mb` | int | 1400 | 1500 | Memory limit (MB) |

**Response:**
- Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- File: `products_export_YYYYMMDD_HHMMSS.xlsx`

**Example:**
```bash
# Export 200k products (default)
curl -o products.xlsx "http://localhost:8080/api/v1/excel/products/export"

# Export 100k products with custom batch size
curl -o products.xlsx "http://localhost:8080/api/v1/excel/products/export?count=100000&batch_size=5000"

# Export với memory limit thấp hơn
curl -o products.xlsx "http://localhost:8080/api/v1/excel/products/export?count=200000&max_memory_mb=1000"
```

### 2. Export Progress (SSE)
```
GET /api/v1/excel/products/export-progress
```

Server-Sent Events stream để theo dõi progress.

**Events:**
- `start`: `{"total": 200000, "message": "Starting export..."}`
- `progress`: `{"processed": 10000, "total": 200000, "percentage": 5.00, "memory_mb": 150}`
- `complete`: `{"message": "Export simulation completed"}`

**Example:**
```javascript
const eventSource = new EventSource('/api/v1/excel/products/export-progress?count=200000');
eventSource.addEventListener('progress', (e) => {
    const data = JSON.parse(e.data);
    console.log(`Progress: ${data.percentage}%, Memory: ${data.memory_mb}MB`);
});
```

### 3. Memory Stats
```
GET /api/v1/excel/memory-stats
```

**Response:**
```json
{
    "alloc_mb": 45,
    "total_alloc_mb": 1200,
    "sys_mb": 200,
    "heap_alloc_mb": 45,
    "heap_sys_mb": 150,
    "heap_idle_mb": 80,
    "heap_inuse_mb": 70,
    "heap_released_mb": 50,
    "heap_objects": 500000,
    "num_gc": 15,
    "gc_cpu_fraction": 0.02
}
```

### 4. Force Garbage Collection
```
POST /api/v1/excel/force-gc
```

**Response:**
```json
{
    "before_mb": 500,
    "after_mb": 150,
    "freed_mb": 350,
    "message": "GC completed"
}
```

## Data Model

### ProductExcel
```go
type ProductExcel struct {
    ID   string
    Name string

    // 8 Image URLs
    ThumbnailImage string
    PrimaryImage   string
    SecondaryImage string
    DetailImage1   string
    DetailImage2   string
    DetailImage3   string
    DetailImage4   string
    DetailImage5   string

    // 13 Metadata fields
    Metadata map[string]string
}
```

### Metadata Fields (13 keys)
1. `brand` - Nike, Adidas, Samsung, Apple...
2. `category` - Electronics, Clothing, Footwear...
3. `subcategory` - Smartphones, Laptops, Shirts...
4. `color` - Red, Blue, Green...
5. `size` - XS, S, M, L, XL...
6. `material` - Cotton, Leather, Aluminum...
7. `weight` - "2.50 kg"
8. `country_of_origin` - USA, China, Japan...
9. `manufacturer` - Foxconn, Samsung Electronics...
10. `sku` - "SKU-Nik-000100"
11. `barcode` - "1000000000100"
12. `warranty_period` - "12 months"
13. `release_date` - "2024-06-15"

### Excel Columns (23 total)
| Column | Field |
|--------|-------|
| A | ID |
| B | Name |
| C | ThumbnailImage |
| D | PrimaryImage |
| E | SecondaryImage |
| F | DetailImage1 |
| G | DetailImage2 |
| H | DetailImage3 |
| I | DetailImage4 |
| J | DetailImage5 |
| K | Meta_brand |
| L | Meta_category |
| M | Meta_subcategory |
| N | Meta_color |
| O | Meta_size |
| P | Meta_material |
| Q | Meta_weight |
| R | Meta_country_of_origin |
| S | Meta_manufacturer |
| T | Meta_sku |
| U | Meta_barcode |
| V | Meta_warranty_period |
| W | Meta_release_date |

## Technical Implementation

### Memory Management Strategy

1. **Streaming Writer**: Sử dụng `excelize.StreamWriter` thay vì write toàn bộ vào memory
2. **On-demand Generation**: Products được generate từng cái, không load 200k vào memory
3. **Periodic GC**: Force garbage collection mỗi 10,000 rows
4. **Memory Monitoring**: Check memory usage mỗi batch, warning nếu > 80% limit
5. **Temp Files**: Excelize tự động dùng temp files khi chunks > 16MB

### Logging Output

```
[Excel Export] Starting export of 200000 products
[Excel Export] Config: BatchSize=10000, MaxMemoryMB=1400, FlushInterval=50000

--- Memory Stats ---
Alloc: 45 MB (heap objects)
TotalAlloc: 45 MB (cumulative)
...

[Excel Export] Step 1 - Create file: 1.234ms
[Excel Export] Step 2 - Create stream writer: 0.5ms
[Excel Export] Step 3 - Set column widths: 0.1ms
[Excel Export] Step 4 - Write header: 0.05ms

[Excel Export] Progress: 10000/200000 (5.00%) | Batch time: 1.2s | Rows/s: 8333 | Memory: 120 MB | Peak: 120 MB
[Excel Export] Progress: 20000/200000 (10.00%) | Batch time: 1.1s | Rows/s: 9090 | Memory: 180 MB | Peak: 180 MB
...

[Excel Export] Step 5 - Generate & write 200000 rows: 25.5s
[Excel Export] Step 6 - Flush stream: 0.8s
[Excel Export] Step 7 - Write to output: 3.2s

=== Excel Export Summary ===
Total Rows: 200000
Processed Rows: 200000
Progress: 100.00%
Total Time: 29.5s
Rows/Second: 6779.66
Peak Memory: 650 MB
Current Memory: 450 MB
Flush Count: 1
GC Count: 20
Average Batch Time: 1.15s
=============================
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| Rows | 200,000 |
| Columns | 23 |
| Total Cells | 4,600,000 |
| Export Time | ~25-35 seconds |
| File Size | ~80-100 MB |
| Peak Memory | ~600-800 MB |
| Rows/Second | ~6,000-8,000 |

## Best Practices

### For Production
1. Đặt timeout đủ dài (30+ minutes)
2. Monitor memory qua `/api/v1/excel/memory-stats`
3. Gọi `/api/v1/excel/force-gc` trước export nếu cần
4. Sử dụng `max_memory_mb` phù hợp với server

### Testing
```bash
# Run tests
go test -v ./internal/service/... -run TestExcel

# Run memory test (slower)
go test -v ./internal/service/... -run TestExcelStreamingExporter_MemoryLimit

# Benchmark
go test -bench=. ./internal/service/... -benchmem
```

## Files Structure

```
internal/
├── domain/
│   └── product_excel.go          # ProductExcel model + ToExcelRow()
├── handler/
│   └── excel_export_handler.go   # HTTP handlers
├── service/
│   ├── product_data_generator.go # Seed data generator
│   ├── excel_streaming_exporter.go      # Main export logic
│   └── excel_streaming_exporter_test.go # Tests
```

## Dependencies

```go
require github.com/xuri/excelize/v2 v2.8.0
```

**Note**: KHÔNG sử dụng thư viện `simpleexcel` trong `pkg/`. Chỉ dùng `excelize/v2` thuần.
