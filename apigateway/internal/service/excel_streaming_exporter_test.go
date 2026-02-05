package service

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"
)

func TestProductDataGenerator_GenerateProduct(t *testing.T) {
	generator := NewProductDataGenerator(42)

	product := generator.GenerateProduct(0)

	// Verify ID format
	if product.ID == "" {
		t.Error("Product ID should not be empty")
	}

	// Verify Name is not empty
	if product.Name == "" {
		t.Error("Product Name should not be empty")
	}

	// Verify all 8 image URLs exist
	if product.ThumbnailImage == "" {
		t.Error("ThumbnailImage should not be empty")
	}
	if product.PrimaryImage == "" {
		t.Error("PrimaryImage should not be empty")
	}
	if product.SecondaryImage == "" {
		t.Error("SecondaryImage should not be empty")
	}
	if product.DetailImage1 == "" {
		t.Error("DetailImage1 should not be empty")
	}
	if product.DetailImage2 == "" {
		t.Error("DetailImage2 should not be empty")
	}
	if product.DetailImage3 == "" {
		t.Error("DetailImage3 should not be empty")
	}
	if product.DetailImage4 == "" {
		t.Error("DetailImage4 should not be empty")
	}
	if product.DetailImage5 == "" {
		t.Error("DetailImage5 should not be empty")
	}

	// Verify Metadata has exactly 13 fields
	if len(product.Metadata) != 13 {
		t.Errorf("Metadata should have 13 fields, got %d", len(product.Metadata))
	}

	// Verify all metadata keys exist
	expectedKeys := []string{
		"brand", "category", "subcategory", "color", "size",
		"material", "weight", "country_of_origin", "manufacturer",
		"sku", "barcode", "warranty_period", "release_date",
	}

	for _, key := range expectedKeys {
		if _, ok := product.Metadata[key]; !ok {
			t.Errorf("Metadata should contain key %q", key)
		}
	}
}

func TestProductDataGenerator_Reproducibility(t *testing.T) {
	// Same seed should produce same results
	gen1 := NewProductDataGenerator(42)
	gen2 := NewProductDataGenerator(42)

	product1 := gen1.GenerateProduct(100)
	product2 := gen2.GenerateProduct(100)

	if product1.ID != product2.ID {
		t.Error("Same seed should produce same ID")
	}
	if product1.Name != product2.Name {
		t.Error("Same seed should produce same Name")
	}
}

func TestProductDataGenerator_Uniqueness(t *testing.T) {
	generator := NewProductDataGenerator(42)

	ids := make(map[string]bool)
	names := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		product := generator.GenerateProduct(i)

		if ids[product.ID] {
			t.Errorf("Duplicate ID found: %s", product.ID)
		}
		ids[product.ID] = true

		if names[product.Name] {
			t.Errorf("Duplicate Name found: %s", product.Name)
		}
		names[product.Name] = true
	}
}

func TestExcelStreamingExporter_SmallExport(t *testing.T) {
	config := ExcelExportConfig{
		TotalProducts: 1000,
		BatchSize:     500,
		MaxMemoryMB:   500,
		EnableLogging: false,
		FlushInterval: 500,
		GCInterval:    500,
	}

	exporter := NewExcelStreamingExporter(config)
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := exporter.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify we got some output
	if buf.Len() == 0 {
		t.Error("Output buffer should not be empty")
	}

	// Check stats
	stats := exporter.GetStats()
	if stats.ProcessedRows != config.TotalProducts {
		t.Errorf("Expected %d processed rows, got %d", config.TotalProducts, stats.ProcessedRows)
	}

	t.Logf("Output size: %d bytes", buf.Len())
	t.Logf("Peak memory: %d MB", stats.PeakMemoryMB)
}

func TestExcelStreamingExporter_MemoryLimit(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	config := ExcelExportConfig{
		TotalProducts: 50000, // 50k for faster test
		BatchSize:     5000,
		MaxMemoryMB:   1400, // Under 1.5GB limit
		EnableLogging: false,
		FlushInterval: 25000,
		GCInterval:    5000,
	}

	exporter := NewExcelStreamingExporter(config)
	var buf bytes.Buffer

	// Force GC before test
	runtime.GC()
	var beforeMem runtime.MemStats
	runtime.ReadMemStats(&beforeMem)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err := exporter.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	stats := exporter.GetStats()

	// Verify memory stayed under limit
	if stats.PeakMemoryMB > uint64(config.MaxMemoryMB) {
		t.Errorf("Peak memory %d MB exceeded limit %d MB", stats.PeakMemoryMB, config.MaxMemoryMB)
	}

	t.Logf("Rows: %d, Output size: %d bytes", stats.ProcessedRows, buf.Len())
	t.Logf("Peak memory: %d MB (limit: %d MB)", stats.PeakMemoryMB, config.MaxMemoryMB)
	t.Logf("GC count: %d", stats.GCCount)
}

func TestGetMemoryUsageMB(t *testing.T) {
	mem := GetMemoryUsageMB()
	if mem == 0 {
		t.Log("Memory usage reported as 0, might be okay for very low usage")
	}
	t.Logf("Current memory usage: %d MB", mem)
}

func TestGetDetailedMemoryStats(t *testing.T) {
	stats := GetDetailedMemoryStats()
	if stats == "" {
		t.Error("Memory stats should not be empty")
	}
	t.Log(stats)
}

func BenchmarkGenerateProduct(b *testing.B) {
	generator := NewProductDataGenerator(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.GenerateProduct(i)
	}
}

func BenchmarkToExcelRow(b *testing.B) {
	generator := NewProductDataGenerator(42)
	product := generator.GenerateProduct(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = product.ToExcelRow()
	}
}

func BenchmarkSmallExport(b *testing.B) {
	config := ExcelExportConfig{
		TotalProducts: 100,
		BatchSize:     50,
		MaxMemoryMB:   500,
		EnableLogging: false,
		FlushInterval: 50,
		GCInterval:    50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exporter := NewExcelStreamingExporter(config)
		var buf bytes.Buffer
		_ = exporter.ExportToWriter(context.Background(), &buf)
	}
}
