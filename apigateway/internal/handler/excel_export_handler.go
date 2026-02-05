package handler

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service"
)

// ExcelExportHandler handles Excel export requests for large datasets
type ExcelExportHandler struct{}

// NewExcelExportHandler creates a new Excel export handler
func NewExcelExportHandler() *ExcelExportHandler {
	return &ExcelExportHandler{}
}

// ExportLargeProductsExcel exports 200,000 products to Excel using streaming
// GET /api/v1/excel/products/export
// Query params:
//   - count: number of products (default: 200000, max: 500000)
//   - batch_size: batch size for logging (default: 10000)
//   - max_memory_mb: max memory in MB (default: 1400)
//   - sections: "true" to enable 3-section export with styles and DIFF formulas
func (h *ExcelExportHandler) ExportLargeProductsExcel(c echo.Context) error {
	ctx := c.Request().Context()
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = fmt.Sprintf("excel-export-%d", time.Now().UnixNano())
	}

	// Log request start
	logger.InfoLog(ctx, "[%s] Excel Export Request Started", requestID)
	logger.InfoLog(ctx, "[%s] Initial Memory: %s", requestID, service.GetDetailedMemoryStats())

	startTime := time.Now()

	// Parse query parameters
	config := service.DefaultExportConfig()
	useSections := c.QueryParam("sections") == "true"

	if countStr := c.QueryParam("count"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil && count > 0 && count <= 500000 {
			config.TotalProducts = count
		}
	}

	if batchStr := c.QueryParam("batch_size"); batchStr != "" {
		if batch, err := strconv.Atoi(batchStr); err == nil && batch > 0 {
			config.BatchSize = batch
		}
	}

	if memStr := c.QueryParam("max_memory_mb"); memStr != "" {
		if mem, err := strconv.Atoi(memStr); err == nil && mem > 0 && mem <= 1500 {
			config.MaxMemoryMB = mem
		}
	}

	logger.InfoLog(ctx, "[%s] Export Config: TotalProducts=%d, BatchSize=%d, MaxMemoryMB=%d, Sections=%v",
		requestID, config.TotalProducts, config.BatchSize, config.MaxMemoryMB, useSections)

	// Create exporter
	exporter := service.NewExcelStreamingExporter(config)

	// Set response headers for file download
	filename := fmt.Sprintf("products_export_%s.xlsx", time.Now().Format("20060102_150405"))
	if useSections {
		filename = fmt.Sprintf("products_3sections_%s.xlsx", time.Now().Format("20060102_150405"))
	}
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")

	// Create context with timeout (generous for large files)
	exportCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// Export to response writer
	c.Response().WriteHeader(http.StatusOK)

	var err error
	if useSections {
		// Export with 3 sections: Original, Readonly Copy, DIFF
		err = exporter.ExportToWriterWithSections(exportCtx, c.Response().Writer)
	} else {
		// Simple export
		err = exporter.ExportToWriter(exportCtx, c.Response().Writer)
	}

	if err != nil {
		logger.ErrorLog(ctx, "[%s] Excel Export Failed: %v", requestID, err)
		// Can't change status code after headers are sent, so just log
		return nil
	}

	// Log completion
	totalTime := time.Since(startTime)
	stats := exporter.GetStats()

	logger.InfoLog(ctx, "[%s] Excel Export Completed Successfully", requestID)
	logger.InfoLog(ctx, "[%s] Total Time: %v", requestID, totalTime)
	logger.InfoLog(ctx, "[%s] Rows Written: %d", requestID, stats.ProcessedRows)
	logger.InfoLog(ctx, "[%s] Peak Memory: %d MB", requestID, stats.PeakMemoryMB)
	logger.InfoLog(ctx, "[%s] Final Memory: %s", requestID, service.GetDetailedMemoryStats())

	// Force final GC
	runtime.GC()

	return nil
}

// ExportProductsWithProgress exports products and streams progress updates via SSE
// GET /api/v1/excel/products/export-progress
func (h *ExcelExportHandler) ExportProductsWithProgress(c echo.Context) error {
	ctx := c.Request().Context()

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Streaming not supported",
		})
	}

	// Parse parameters
	config := service.DefaultExportConfig()
	if countStr := c.QueryParam("count"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil && count > 0 && count <= 500000 {
			config.TotalProducts = count
		}
	}

	// Send start event
	fmt.Fprintf(c.Response().Writer, "event: start\n")
	fmt.Fprintf(c.Response().Writer, "data: {\"total\": %d, \"message\": \"Starting export...\"}\n\n", config.TotalProducts)
	flusher.Flush()

	// Create a goroutine to generate products and track progress
	generator := service.NewProductDataGenerator(42)
	progressChan := make(chan int, 100)

	go func() {
		defer close(progressChan)
		for i := 0; i < config.TotalProducts; i++ {
			_ = generator.GenerateProduct(i)
			if (i+1)%10000 == 0 {
				progressChan <- i + 1
			}
		}
		progressChan <- config.TotalProducts
	}()

	// Stream progress updates
	for progress := range progressChan {
		select {
		case <-ctx.Done():
			return nil
		default:
			memMB := service.GetMemoryUsageMB()
			percentage := float64(progress) / float64(config.TotalProducts) * 100

			fmt.Fprintf(c.Response().Writer, "event: progress\n")
			fmt.Fprintf(c.Response().Writer, "data: {\"processed\": %d, \"total\": %d, \"percentage\": %.2f, \"memory_mb\": %d}\n\n",
				progress, config.TotalProducts, percentage, memMB)
			flusher.Flush()
		}
	}

	// Send complete event
	fmt.Fprintf(c.Response().Writer, "event: complete\n")
	fmt.Fprintf(c.Response().Writer, "data: {\"message\": \"Export simulation completed\"}\n\n")
	flusher.Flush()

	logger.InfoLog(ctx, "[SSE Export] Progress streaming completed")

	return nil
}

// GetMemoryStats returns current memory statistics
// GET /api/v1/excel/memory-stats
func (h *ExcelExportHandler) GetMemoryStats(c echo.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"alloc_mb":         m.Alloc / 1024 / 1024,
		"total_alloc_mb":   m.TotalAlloc / 1024 / 1024,
		"sys_mb":           m.Sys / 1024 / 1024,
		"heap_alloc_mb":    m.HeapAlloc / 1024 / 1024,
		"heap_sys_mb":      m.HeapSys / 1024 / 1024,
		"heap_idle_mb":     m.HeapIdle / 1024 / 1024,
		"heap_inuse_mb":    m.HeapInuse / 1024 / 1024,
		"heap_released_mb": m.HeapReleased / 1024 / 1024,
		"heap_objects":     m.HeapObjects,
		"num_gc":           m.NumGC,
		"gc_cpu_fraction":  m.GCCPUFraction,
	}

	return c.JSON(http.StatusOK, stats)
}

// ForceGC triggers garbage collection
// POST /api/v1/excel/force-gc
func (h *ExcelExportHandler) ForceGC(c echo.Context) error {
	ctx := c.Request().Context()

	beforeMem := service.GetMemoryUsageMB()
	logger.InfoLog(ctx, "[Force GC] Memory before: %d MB", beforeMem)

	runtime.GC()

	afterMem := service.GetMemoryUsageMB()
	freed := int64(beforeMem) - int64(afterMem)

	logger.InfoLog(ctx, "[Force GC] Memory after: %d MB, freed: %d MB", afterMem, freed)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"before_mb": beforeMem,
		"after_mb":  afterMem,
		"freed_mb":  freed,
		"message":   "GC completed",
	})
}
