package service

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/xuri/excelize/v2"
)

// ExcelExportConfig contains configuration for Excel export
type ExcelExportConfig struct {
	TotalProducts     int           // Total number of products to export
	BatchSize         int           // Number of products per batch for memory management
	MaxMemoryMB       int           // Maximum memory usage in MB (default: 1500)
	EnableLogging     bool          // Enable detailed logging
	FlushInterval     int           // Flush to disk every N rows
	GCInterval        int           // Force GC every N rows
	MemoryCheckPeriod time.Duration // Period between memory checks
}

// DefaultExportConfig returns sensible defaults for large file export
func DefaultExportConfig() ExcelExportConfig {
	return ExcelExportConfig{
		TotalProducts:     200000,
		BatchSize:         10000,
		MaxMemoryMB:       1400, // Keep some buffer below 1.5GB
		EnableLogging:     true,
		FlushInterval:     50000, // Flush every 50k rows
		GCInterval:        10000, // GC every 10k rows
		MemoryCheckPeriod: 5 * time.Second,
	}
}

// ExcelExportStats tracks export statistics
type ExcelExportStats struct {
	mu sync.RWMutex

	StartTime           time.Time
	EndTime             time.Time
	TotalRows           int
	ProcessedRows       int
	CurrentMemoryMB     uint64
	PeakMemoryMB        uint64
	FlushCount          int
	GCCount             int
	BatchProcessingTime []time.Duration
}

func NewExcelExportStats() *ExcelExportStats {
	return &ExcelExportStats{
		StartTime:           time.Now(),
		BatchProcessingTime: make([]time.Duration, 0),
	}
}

func (s *ExcelExportStats) UpdateMemory(currentMB uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentMemoryMB = currentMB
	if currentMB > s.PeakMemoryMB {
		s.PeakMemoryMB = currentMB
	}
}

func (s *ExcelExportStats) IncrementProcessed(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ProcessedRows += count
}

func (s *ExcelExportStats) AddBatchTime(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BatchProcessingTime = append(s.BatchProcessingTime, d)
}

func (s *ExcelExportStats) GetProgress() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.TotalRows == 0 {
		return 0
	}
	return float64(s.ProcessedRows) / float64(s.TotalRows) * 100
}

func (s *ExcelExportStats) GetSummary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalTime := time.Since(s.StartTime)
	avgBatchTime := time.Duration(0)
	if len(s.BatchProcessingTime) > 0 {
		var total time.Duration
		for _, d := range s.BatchProcessingTime {
			total += d
		}
		avgBatchTime = total / time.Duration(len(s.BatchProcessingTime))
	}

	rowsPerSec := float64(s.ProcessedRows) / totalTime.Seconds()

	return fmt.Sprintf(`
=== Excel Export Summary ===
Total Rows: %d
Processed Rows: %d
Progress: %.2f%%
Total Time: %v
Rows/Second: %.2f
Peak Memory: %d MB
Current Memory: %d MB
Flush Count: %d
GC Count: %d
Average Batch Time: %v
=============================`,
		s.TotalRows,
		s.ProcessedRows,
		s.GetProgress(),
		totalTime,
		rowsPerSec,
		s.PeakMemoryMB,
		s.CurrentMemoryMB,
		s.FlushCount,
		s.GCCount,
		avgBatchTime,
	)
}

// ExcelStreamingExporter exports large datasets to Excel using streaming
type ExcelStreamingExporter struct {
	config ExcelExportConfig
	stats  *ExcelExportStats
}

func NewExcelStreamingExporter(config ExcelExportConfig) *ExcelStreamingExporter {
	return &ExcelStreamingExporter{
		config: config,
		stats:  NewExcelExportStats(),
	}
}

// GetMemoryUsageMB returns current memory usage in MB
func GetMemoryUsageMB() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc / 1024 / 1024
}

// GetDetailedMemoryStats returns detailed memory statistics
func GetDetailedMemoryStats() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf(`
--- Memory Stats ---
Alloc: %d MB (heap objects)
TotalAlloc: %d MB (cumulative)
Sys: %d MB (system memory)
HeapAlloc: %d MB
HeapSys: %d MB
HeapIdle: %d MB
HeapInuse: %d MB
HeapReleased: %d MB
HeapObjects: %d
StackSys: %d MB
GCSys: %d MB
NumGC: %d
-------------------`,
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Sys/1024/1024,
		m.HeapAlloc/1024/1024,
		m.HeapSys/1024/1024,
		m.HeapIdle/1024/1024,
		m.HeapInuse/1024/1024,
		m.HeapReleased/1024/1024,
		m.HeapObjects,
		m.StackSys/1024/1024,
		m.GCSys/1024/1024,
		m.NumGC,
	)
}

// ExportToWriter exports products to an io.Writer using streaming
// This is the main export function that handles memory management
func (e *ExcelStreamingExporter) ExportToWriter(ctx context.Context, writer io.Writer) error {
	e.stats = NewExcelExportStats()
	e.stats.TotalRows = e.config.TotalProducts

	// Log initial state
	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Starting export of %d products", e.config.TotalProducts)
		logger.InfoLog(ctx, "[Excel Export] Config: BatchSize=%d, MaxMemoryMB=%d, FlushInterval=%d",
			e.config.BatchSize, e.config.MaxMemoryMB, e.config.FlushInterval)
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	// Step 1: Create new Excel file
	stepStart := time.Now()
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			logger.ErrorLog(ctx, "[Excel Export] Error closing file: %v", err)
		}
	}()

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 1 - Create file: %v", time.Since(stepStart))
	}

	// Step 2: Get stream writer
	stepStart = time.Now()
	sheetName := "Products"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// Delete default Sheet1 if it exists
	if err := f.DeleteSheet("Sheet1"); err != nil {
		logger.WarnLog(ctx, "[Excel Export] Could not delete Sheet1: %v", err)
	}

	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create stream writer: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 2 - Create stream writer: %v", time.Since(stepStart))
	}

	// Step 3: Set column widths (must be before writing rows)
	stepStart = time.Now()
	columnWidths := []float64{
		15, // ID
		40, // Name
		50, // ThumbnailImage
		50, // PrimaryImage
		50, // SecondaryImage
		50, // DetailImage1
		50, // DetailImage2
		50, // DetailImage3
		50, // DetailImage4
		50, // DetailImage5
		15, // Meta_brand
		15, // Meta_category
		15, // Meta_subcategory
		12, // Meta_color
		10, // Meta_size
		15, // Meta_material
		10, // Meta_weight
		20, // Meta_country_of_origin
		25, // Meta_manufacturer
		20, // Meta_sku
		15, // Meta_barcode
		15, // Meta_warranty_period
		15, // Meta_release_date
	}

	for i, width := range columnWidths {
		if err := sw.SetColWidth(i+1, i+1, width); err != nil {
			return fmt.Errorf("failed to set column width %d: %w", i, err)
		}
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 3 - Set column widths: %v", time.Since(stepStart))
	}

	// Step 4: Write header row
	stepStart = time.Now()
	headers := domain.GetExcelHeaders()
	headerRow := make([]interface{}, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}

	if err := sw.SetRow("A1", headerRow); err != nil {
		return fmt.Errorf("failed to write header row: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 4 - Write header: %v", time.Since(stepStart))
	}

	// Step 5: Generate and write data rows
	stepStart = time.Now()
	generator := NewProductDataGenerator(42) // Fixed seed for reproducibility

	rowNum := 2 // Start from row 2 (row 1 is header)
	batchStart := time.Now()

	for i := 0; i < e.config.TotalProducts; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Generate product
		product := generator.GenerateProduct(i)

		// Write row
		cell, err := excelize.CoordinatesToCellName(1, rowNum)
		if err != nil {
			return fmt.Errorf("failed to get cell name for row %d: %w", rowNum, err)
		}

		if err := sw.SetRow(cell, product.ToExcelRow()); err != nil {
			return fmt.Errorf("failed to write row %d: %w", rowNum, err)
		}

		rowNum++
		e.stats.ProcessedRows = i + 1

		// Periodic memory check and GC
		if (i+1)%e.config.GCInterval == 0 {
			currentMem := GetMemoryUsageMB()
			e.stats.UpdateMemory(currentMem)

			// Force GC if approaching memory limit
			if currentMem > uint64(e.config.MaxMemoryMB*80/100) {
				runtime.GC()
				e.stats.GCCount++

				if e.config.EnableLogging {
					logger.WarnLog(ctx, "[Excel Export] Memory pressure at row %d: %d MB, forcing GC",
						i+1, currentMem)
				}
			}

			// Check if we're over the limit
			afterGC := GetMemoryUsageMB()
			if afterGC > uint64(e.config.MaxMemoryMB) {
				logger.ErrorLog(ctx, "[Excel Export] Memory limit exceeded: %d MB > %d MB",
					afterGC, e.config.MaxMemoryMB)
				// Don't fail, but log the warning
			}
		}

		// Log progress
		if e.config.EnableLogging && (i+1)%e.config.BatchSize == 0 {
			batchTime := time.Since(batchStart)
			e.stats.AddBatchTime(batchTime)

			currentMem := GetMemoryUsageMB()
			e.stats.UpdateMemory(currentMem)

			rowsPerSec := float64(e.config.BatchSize) / batchTime.Seconds()
			progress := float64(i+1) / float64(e.config.TotalProducts) * 100

			logger.InfoLog(ctx,
				"[Excel Export] Progress: %d/%d (%.2f%%) | Batch time: %v | Rows/s: %.0f | Memory: %d MB | Peak: %d MB",
				i+1, e.config.TotalProducts, progress, batchTime, rowsPerSec,
				currentMem, e.stats.PeakMemoryMB,
			)

			batchStart = time.Now()
		}
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 5 - Generate & write %d rows: %v",
			e.config.TotalProducts, time.Since(stepStart))
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	// Step 6: Flush stream writer
	stepStart = time.Now()
	if err := sw.Flush(); err != nil {
		return fmt.Errorf("failed to flush stream writer: %w", err)
	}
	e.stats.FlushCount++

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 6 - Flush stream: %v", time.Since(stepStart))
	}

	// Force GC before final write
	runtime.GC()
	e.stats.GCCount++

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Post-flush GC complete")
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	// Step 7: Write to output
	stepStart = time.Now()
	if _, err := f.WriteTo(writer); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 7 - Write to output: %v", time.Since(stepStart))
	}

	// Final stats
	e.stats.EndTime = time.Now()
	if e.config.EnableLogging {
		logger.InfoLog(ctx, "%s", e.stats.GetSummary())
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	return nil
}

// GetStats returns current export statistics
func (e *ExcelStreamingExporter) GetStats() *ExcelExportStats {
	return e.stats
}
