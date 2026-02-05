package service

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/xuri/excelize/v2"
)

// Section column counts
const (
	NumColumns       = 23 // Total columns per section (ID + Name + 8 Images + 13 Metadata)
	Section1StartCol = 1  // Column A
	Section2StartCol = 24 // Column X (after Section 1)
	Section3StartCol = 47 // Column AU (after Section 2)
)

// ExcelStyles holds all style IDs for the export
type ExcelStyles struct {
	// Header styles
	HeaderSection1  int // Blue for Section 1 headers
	HeaderSection2  int // Green for Section 2 headers (readonly)
	HeaderSection3  int // Orange for Section 3 headers (diff)
	HeaderImageCols int // Purple for image column headers

	// Data styles
	DataDefault   int // Default data style
	DataLocked    int // Locked cells for Section 2
	DataDiffMatch int // Green background when values match
	DataDiffError int // Red background when values differ
}

// createExcelStyles creates all styles needed for the export
func createExcelStyles(f *excelize.File) (*ExcelStyles, error) {
	styles := &ExcelStyles{}
	var err error

	// Header Section 1 - Blue background, white bold text
	styles.HeaderSection1, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
			Size:  11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HeaderSection1 style: %w", err)
	}

	// Header Section 2 - Green background (readonly indicator)
	styles.HeaderSection2, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"70AD47"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
			Size:  11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HeaderSection2 style: %w", err)
	}

	// Header Section 3 - Orange background (diff section)
	styles.HeaderSection3, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"ED7D31"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
			Size:  11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HeaderSection3 style: %w", err)
	}

	// Header for Image columns - Purple
	styles.HeaderImageCols, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"7030A0"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
			Size:  11,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HeaderImageCols style: %w", err)
	}

	// Default data style
	styles.DataDefault, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: false,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DataDefault style: %w", err)
	}

	// Locked data style (for Section 2)
	styles.DataLocked, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"E2EFDA"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: false,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "A9D08E", Style: 1},
			{Type: "right", Color: "A9D08E", Style: 1},
			{Type: "top", Color: "A9D08E", Style: 1},
			{Type: "bottom", Color: "A9D08E", Style: 1},
		},
		Protection: &excelize.Protection{
			Locked: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DataLocked style: %w", err)
	}

	// Diff Match style - Light green
	styles.DataDiffMatch, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"C6EFCE"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "006100",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "006100", Style: 1},
			{Type: "right", Color: "006100", Style: 1},
			{Type: "top", Color: "006100", Style: 1},
			{Type: "bottom", Color: "006100", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DataDiffMatch style: %w", err)
	}

	// Diff Error style - Light red
	styles.DataDiffError, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFC7CE"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "9C0006",
			Bold:  true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "9C0006", Style: 1},
			{Type: "right", Color: "9C0006", Style: 1},
			{Type: "top", Color: "9C0006", Style: 1},
			{Type: "bottom", Color: "9C0006", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DataDiffError style: %w", err)
	}

	return styles, nil
}

// ExportToWriterWithSections exports products with 3 sections:
// Section 1: Original data (editable)
// Section 2: Copy of data (locked/readonly)
// Section 3: DIFF comparison formulas
func (e *ExcelStreamingExporter) ExportToWriterWithSections(ctx context.Context, writer io.Writer) error {
	e.stats = NewExcelExportStats()
	e.stats.TotalRows = e.config.TotalProducts

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Starting 3-section export of %d products", e.config.TotalProducts)
		logger.InfoLog(ctx, "[Excel Export] Config: BatchSize=%d, MaxMemoryMB=%d", e.config.BatchSize, e.config.MaxMemoryMB)
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

	// Step 2: Create styles
	stepStart = time.Now()
	styles, err := createExcelStyles(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 2 - Create styles: %v", time.Since(stepStart))
	}

	// Step 3: Setup sheet
	stepStart = time.Now()
	sheetName := "Products"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)

	if err := f.DeleteSheet("Sheet1"); err != nil {
		logger.WarnLog(ctx, "[Excel Export] Could not delete Sheet1: %v", err)
	}

	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create stream writer: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 3 - Create stream writer: %v", time.Since(stepStart))
	}

	// Step 4: Set column widths for all 3 sections
	stepStart = time.Now()
	baseColumnWidths := []float64{
		15, // ID
		40, // Name
		35, // ThumbnailImage
		35, // PrimaryImage
		35, // SecondaryImage
		35, // DetailImage1
		35, // DetailImage2
		35, // DetailImage3
		35, // DetailImage4
		35, // DetailImage5
		12, // Meta_brand
		12, // Meta_category
		15, // Meta_subcategory
		10, // Meta_color
		8,  // Meta_size
		12, // Meta_material
		10, // Meta_weight
		18, // Meta_country_of_origin
		20, // Meta_manufacturer
		18, // Meta_sku
		15, // Meta_barcode
		15, // Meta_warranty_period
		12, // Meta_release_date
	}

	// Set widths for all 3 sections
	for section := 0; section < 3; section++ {
		startCol := 1 + section*NumColumns
		for i, width := range baseColumnWidths {
			colNum := startCol + i
			// Section 3 (diff) columns are narrower
			actualWidth := width
			if section == 2 {
				actualWidth = 10 // Diff columns just show "OK" or "DIFF"
			}
			if err := sw.SetColWidth(colNum, colNum, actualWidth); err != nil {
				return fmt.Errorf("failed to set column width %d: %w", colNum, err)
			}
		}
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 4 - Set column widths for 3 sections: %v", time.Since(stepStart))
	}

	// Step 5: Write header rows for all 3 sections
	stepStart = time.Now()
	headers := domain.GetExcelHeaders()

	// Build complete header row for all 3 sections
	totalCols := NumColumns * 3
	headerRow := make([]interface{}, totalCols)

	// Section 1 headers (with styles)
	for i, h := range headers {
		style := styles.HeaderSection1
		// Image columns (index 2-9) get purple style
		if i >= 2 && i <= 9 {
			style = styles.HeaderImageCols
		}
		headerRow[i] = excelize.Cell{
			Value:   h,
			StyleID: style,
		}
	}

	// Section 2 headers (READONLY prefix)
	for i, h := range headers {
		style := styles.HeaderSection2
		if i >= 2 && i <= 9 {
			style = styles.HeaderImageCols
		}
		headerRow[NumColumns+i] = excelize.Cell{
			Value:   "[RO] " + h, // ReadOnly prefix
			StyleID: style,
		}
	}

	// Section 3 headers (DIFF prefix)
	for i, h := range headers {
		headerRow[NumColumns*2+i] = excelize.Cell{
			Value:   "Δ " + h, // Delta symbol for diff
			StyleID: styles.HeaderSection3,
		}
	}

	if err := sw.SetRow("A1", headerRow); err != nil {
		return fmt.Errorf("failed to write header row: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 5 - Write headers for 3 sections: %v", time.Since(stepStart))
	}

	// Step 6: Generate and write data rows
	stepStart = time.Now()
	generator := NewProductDataGenerator(42)

	rowNum := 2
	batchStart := time.Now()

	for i := 0; i < e.config.TotalProducts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		product := generator.GenerateProduct(i)
		productData := product.ToExcelRow()

		// Build complete row for all 3 sections
		fullRow := make([]interface{}, totalCols)

		// Section 1: Original data (editable)
		for j, val := range productData {
			fullRow[j] = excelize.Cell{
				Value:   val,
				StyleID: styles.DataDefault,
			}
		}

		// Section 2: Copy of data (locked/readonly style)
		for j, val := range productData {
			fullRow[NumColumns+j] = excelize.Cell{
				Value:   val,
				StyleID: styles.DataLocked,
			}
		}

		// Section 3: DIFF formulas comparing Section 1 and Section 2
		for j := 0; j < NumColumns; j++ {
			// Get cell references for Section 1 and Section 2
			sec1Cell, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			sec2Cell, _ := excelize.CoordinatesToCellName(NumColumns+j+1, rowNum)

			// Formula: IF(A2=X2, "OK", "DIFF")
			formula := fmt.Sprintf(`IF(%s=%s,"✓ OK","✗ DIFF")`, sec1Cell, sec2Cell)

			fullRow[NumColumns*2+j] = excelize.Cell{
				Formula: formula,
				StyleID: styles.DataDefault,
			}
		}

		cell, err := excelize.CoordinatesToCellName(1, rowNum)
		if err != nil {
			return fmt.Errorf("failed to get cell name for row %d: %w", rowNum, err)
		}

		if err := sw.SetRow(cell, fullRow); err != nil {
			return fmt.Errorf("failed to write row %d: %w", rowNum, err)
		}

		rowNum++
		e.stats.ProcessedRows = i + 1

		// Memory management
		if (i+1)%e.config.GCInterval == 0 {
			currentMem := GetMemoryUsageMB()
			e.stats.UpdateMemory(currentMem)

			if currentMem > uint64(e.config.MaxMemoryMB*80/100) {
				runtime.GC()
				e.stats.GCCount++

				if e.config.EnableLogging {
					logger.WarnLog(ctx, "[Excel Export] Memory pressure at row %d: %d MB, forcing GC", i+1, currentMem)
				}
			}
		}

		// Progress logging
		if e.config.EnableLogging && (i+1)%e.config.BatchSize == 0 {
			batchTime := time.Since(batchStart)
			e.stats.AddBatchTime(batchTime)

			currentMem := GetMemoryUsageMB()
			e.stats.UpdateMemory(currentMem)

			rowsPerSec := float64(e.config.BatchSize) / batchTime.Seconds()
			progress := float64(i+1) / float64(e.config.TotalProducts) * 100

			logger.InfoLog(ctx,
				"[Excel Export] Progress: %d/%d (%.2f%%) | Batch: %v | Rows/s: %.0f | Mem: %d MB",
				i+1, e.config.TotalProducts, progress, batchTime, rowsPerSec, currentMem)

			batchStart = time.Now()
		}
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 6 - Generate & write %d rows (3 sections): %v",
			e.config.TotalProducts, time.Since(stepStart))
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	// Step 7: Flush stream writer
	stepStart = time.Now()
	if err := sw.Flush(); err != nil {
		return fmt.Errorf("failed to flush stream writer: %w", err)
	}
	e.stats.FlushCount++

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 7 - Flush stream: %v", time.Since(stepStart))
	}

	// Step 8: Set freeze panes (freeze first 10 columns - ID to DetailImage5)
	stepStart = time.Now()
	if err := f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      10, // Freeze columns A-J (ID to DetailImage5)
		YSplit:      1,  // Freeze header row
		TopLeftCell: "K2",
		ActivePane:  "bottomRight",
	}); err != nil {
		logger.WarnLog(ctx, "[Excel Export] Could not set freeze panes: %v", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 8 - Set freeze panes: %v", time.Since(stepStart))
	}

	// Step 9: Add conditional formatting for Section 3 (DIFF column)
	stepStart = time.Now()
	lastRow := e.config.TotalProducts + 1

	// Range for Section 3
	sec3StartCell, _ := excelize.CoordinatesToCellName(Section3StartCol, 2)
	sec3EndCell, _ := excelize.CoordinatesToCellName(Section3StartCol+NumColumns-1, lastRow)
	sec3Range := fmt.Sprintf("%s:%s", sec3StartCell, sec3EndCell)

	// Green for "OK"
	greenFormat, _ := f.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"C6EFCE"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "006100",
		},
	})

	// Red for "DIFF"
	redFormat, _ := f.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFC7CE"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Color: "9C0006",
			Bold:  true,
		},
	})

	if err := f.SetConditionalFormat(sheetName, sec3Range, []excelize.ConditionalFormatOptions{
		{
			Type:     "cell",
			Criteria: "containing",
			Format:   greenFormat,
			Value:    `"OK"`,
		},
		{
			Type:     "cell",
			Criteria: "containing",
			Format:   redFormat,
			Value:    `"DIFF"`,
		},
	}); err != nil {
		logger.WarnLog(ctx, "[Excel Export] Could not set conditional formatting: %v", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 9 - Set conditional formatting: %v", time.Since(stepStart))
	}

	// Step 10: Protect Section 2 columns (make readonly)
	stepStart = time.Now()
	// Note: Full protection requires sheet protection, but we're using visual styling
	// to indicate readonly. Full protection would prevent all edits.

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 10 - Section 2 visual protection applied: %v", time.Since(stepStart))
	}

	// Force GC before final write
	runtime.GC()
	e.stats.GCCount++

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Post-flush GC complete")
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	// Step 11: Write to output
	stepStart = time.Now()
	if _, err := f.WriteTo(writer); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	if e.config.EnableLogging {
		logger.InfoLog(ctx, "[Excel Export] Step 11 - Write to output: %v", time.Since(stepStart))
	}

	e.stats.EndTime = time.Now()
	if e.config.EnableLogging {
		logger.InfoLog(ctx, "%s", e.stats.GetSummary())
		logger.InfoLog(ctx, "%s", GetDetailedMemoryStats())
	}

	return nil
}
