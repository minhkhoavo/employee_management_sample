package simpleexcelv2

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// Streamer manages a streaming export session.
type Streamer struct {
	exporter *ExcelDataExporter
	file     *excelize.File
	writer   io.Writer
	// streamWriters holds active stream writers for each sheet
	streamWriters map[string]*excelize.StreamWriter
	// currentSheetIndex tracks which sheet we are currently processing
	currentSheetIndex int
	// currentSectionIndex tracks which section we are in within the current sheet
	currentSectionIndex int
	// currentRow keeps track of the current row number for the current sheet
	currentRow int
	// sectionStarted indicates whether the current section's title/header has been written
	sectionStarted bool
}

// Write appends a batch of data to the specified section.
// The sectionID must match the ID of the current section or a future section.
// Strict ordering is enforced: you must write to sections in the order they are defined.
func (s *Streamer) Write(sectionID string, data interface{}) error {
	// 1. Validation
	if s.file == nil {
		return fmt.Errorf("stream is closed or not initialized")
	}

	sheet := s.getCurrentSheet()
	if sheet == nil {
		return fmt.Errorf("no active sheet to write to")
	}

	// Find the target section index
	targetIndex := -1
	for i := s.currentSectionIndex; i < len(sheet.sections); i++ {
		if sheet.sections[i].ID == sectionID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Errorf("section '%s' not found in remaining sections of sheet '%s' (already passed or does not exist)", sectionID, sheet.name)
	}

	sw := s.streamWriters[sheet.name]

	// 2. Advance if needed
	if targetIndex > s.currentSectionIndex {
		// We are moving to a new section.
		// Iterate through sections we are leaving/skipping.
		for i := s.currentSectionIndex; i < targetIndex; i++ {
			sec := sheet.sections[i]
			// If we are leaving the current section and we already started it (wrote Title/Header),
			// we don't need to do anything (data provided manually).
			// If we skipped it (sectionStarted == false), we render it as static (Title/Header only potentially).
			if i == s.currentSectionIndex && s.sectionStarted {
				// Just leaving.
			} else {
				// Skipping or Static section.
				if err := s.renderStaticSection(sw, sec); err != nil {
					return err
				}
			}
		}
		s.currentSectionIndex = targetIndex
		s.sectionStarted = false
	}

	// 3. Current Section
	sec := sheet.sections[s.currentSectionIndex]

	// 4. Resolve Columns (once if not done)
	initialWrite := false
	if len(sec.Columns) == 0 || (len(sec.Columns) > 0 && len(sec.Columns[0].FieldName) == 0) {
		// Dynamic discovery needed
		sec.Columns = mergeColumns(data, sec.Columns)
		initialWrite = true
	} else if !s.sectionStarted {
		// Columns exist but we haven't started this section (haven't written title/header)
		initialWrite = true
	}

	// 5. Render Title & Header (Lazy)
	if initialWrite {
		s.sectionStarted = true

		// Render Title
		if sec.Title != nil {
			cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
			defaultTitleOnly := &StyleTemplate{
				Font:      &FontTemplate{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			styleTmpl := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
			sid, err := s.exporter.createStyle(s.file, styleTmpl)
			if err != nil {
				return err
			}

			colSpan := sec.ColSpan
			if colSpan <= 0 {
				colSpan = len(sec.Columns)
			}
			if colSpan < 1 {
				colSpan = 1
			}

			if err := sw.SetRow(cell, []interface{}{
				excelize.Cell{Value: sec.Title, StyleID: sid},
			}); err != nil {
				return err
			}
			if colSpan > 1 {
				endCell, _ := excelize.CoordinatesToCellName(colSpan, s.currentRow)
				sw.MergeCell(cell, endCell)
			}
			s.currentRow++
		}

		// Render Header
		if sec.ShowHeader && len(sec.Columns) > 0 {
			cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
			headers := make([]interface{}, len(sec.Columns))
			for i, col := range sec.Columns {
				defaultHeader := &StyleTemplate{
					Font:      &FontTemplate{Bold: true},
					Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
				}
				styleTmpl := resolveStyle(sec.HeaderStyle, defaultHeader, col.IsLocked(sec.Locked))
				sid, err := s.exporter.createStyle(s.file, styleTmpl)
				if err != nil {
					return err
				}
				headers[i] = excelize.Cell{Value: col.Header, StyleID: sid}
				if col.Width > 0 {
					sw.SetColWidth(i+1, i+1, col.Width)
				}
			}
			if err := sw.SetRow(cell, headers); err != nil {
				return err
			}
			s.currentRow++
		}
	}

	// 6. Write Data Rows
	return s.writeBatch(sw, sec, data)
}

// Close finishes the specified section (if any active) and moves to the next.
// If the section is "stream-data", this signals end of data for that section.
// Actually, `Write` keeps writing to the same section.
// Use `NextSection()` or `Finish()`?
// Simplest API: Write(id, data)... Write(id, data)... -> then Write(nextID, data).
// Logic: If user calls Write with a DIFFERENT ID, we assume previous is done.
// What if next section is static? We should have likely auto-skipped it.
//
// Let's refine `Write` logic.
// user: Write("sec1", batch1)
// user: Write("sec1", batch2)
// user: Write("sec2", batch3) -> implicitly closes sec1, advances to sec2.

// We need a helper to check if we need to advance.
// But `Write` takes `sectionID`.

// Close finishes the stream and writes the file to the output.
func (s *Streamer) Close() error {
	// Finish current sheet
	if err := s.finishCurrentSheet(); err != nil {
		return err
	}

	// Flush all stream writers
	for _, sw := range s.streamWriters {
		if err := sw.Flush(); err != nil {
			return err
		}
	}

	// Write entire file to output
	if _, err := s.file.WriteTo(s.writer); err != nil {
		return err
	}

	return nil
}

// finishCurrentSheet finishes processing the current sheet (render remaining static sections)
func (s *Streamer) finishCurrentSheet() error {
	// Process remaining sections in current sheet
	sheet := s.getCurrentSheet()
	if sheet == nil {
		return nil
	}

	for s.currentSectionIndex < len(sheet.sections) {
		// If we are here, it means we are closing the sheet.
		// Any remaining sections must be static or empty.
		// If they have ID but no data provided via Write, hopefully they have bound data?
		// We just try to render them as static.

		// Note: We need to advance section index to avoid infinite loop
		// But renderStaticSection might do it?
		// Let's use advanceToNextStreamingSection logic which skips statics.
		idxStart := s.currentSectionIndex
		if err := s.advanceToNextStreamingSection(); err != nil {
			return err
		}
		if s.currentSectionIndex == idxStart {
			// Didn't move? Force move to avoid loop if we are stuck on a streaming section that got no data?
			s.currentSectionIndex++
		}
	}
	return nil
}

func (s *Streamer) getCurrentSheet() *SheetBuilder {
	if s.currentSheetIndex >= len(s.exporter.sheets) {
		return nil
	}
	return s.exporter.sheets[s.currentSheetIndex]
}

// advanceToNextStreamingSection renders all static sections until it hits a section
// that expects streaming data (ID present, Data nil) or end of sheet.
// Returns nil if successful.
func (s *Streamer) advanceToNextStreamingSection() error {
	sheet := s.getCurrentSheet()
	if sheet == nil {
		return nil
	}

	sw := s.streamWriters[sheet.name]

	for s.currentSectionIndex < len(sheet.sections) {
		sec := sheet.sections[s.currentSectionIndex]

		// Is this section waiting for stream data?
		// Criteria: Has ID, and NO Data bound in exporter.
		// If Data is already bound, it's static.
		isStatic := false
		if sec.Data != nil {
			isStatic = true
		} else if sec.ID != "" {
			if data, ok := s.exporter.data[sec.ID]; ok {
				sec.Data = data
				isStatic = true
			}
		} else {
			// No ID? Must be static (e.g. Title only, or static config)
			isStatic = true
		}

		if !isStatic {
			// Found a streaming section! Stop here.
			// User must call Write(sec.ID, ...) next.
			s.sectionStarted = false
			return nil
		}

		// Render Static Section
		if err := s.renderStaticSection(sw, sec); err != nil {
			return err
		}

		s.currentSectionIndex++
	}

	// If we reached end of sheet, move to next sheet?
	// For now, let's keep it simple. User might manually switch?
	// Or we auto-advance to next sheet if this one is done.
	if s.currentSectionIndex >= len(sheet.sections) {
		s.currentSheetIndex++
		s.currentSectionIndex = 0
		s.currentRow = 1
		// Recursively advance in next sheet
		return s.advanceToNextStreamingSection()
	}

	return nil
}

func (s *Streamer) renderStaticSection(sw *excelize.StreamWriter, sec *SectionConfig) error {
	// Re-use logic from renderSections?
	// renderSections is built for "File" API (SetCellValue).
	// StreamWriter API (SetRow) is different.
	// We must duplicate some logic or adapt it.
	// Given strict constraints, we'll reimplement simplified version for Stream.

	// 1. Title
	if sec.Title != nil {
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
		// Title Style
		defaultTitleOnly := &StyleTemplate{
			Font:      &FontTemplate{Bold: true},
			Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
		}
		styleTmpl := resolveStyle(sec.TitleStyle, defaultTitleOnly, sec.Locked)
		sid, err := s.exporter.createStyle(s.file, styleTmpl)
		if err != nil {
			return err
		}

		// Merge logic? StreamWriter doesn't support MergeCell easily in flow?
		// Actually it does: sw.MergeCell(hCell, vCell)
		// But usually done after? No, can be done anytime.

		colSpan := sec.ColSpan
		if colSpan <= 0 {
			colSpan = len(sec.Columns)
		}
		if colSpan < 1 {
			colSpan = 1
		} // unexpected

		// Write Title
		if err := sw.SetRow(cell, []interface{}{
			excelize.Cell{Value: sec.Title, StyleID: sid},
		}); err != nil {
			return err
		}

		if colSpan > 1 {
			endCell, _ := excelize.CoordinatesToCellName(colSpan, s.currentRow)
			sw.MergeCell(cell, endCell)
		}
		s.currentRow++
	}

	// 2. Header
	if sec.ShowHeader && len(sec.Columns) > 0 {
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)

		headers := make([]interface{}, len(sec.Columns))
		for i, col := range sec.Columns {
			defaultHeader := &StyleTemplate{
				Font:      &FontTemplate{Bold: true},
				Alignment: &AlignmentTemplate{Horizontal: "center", Vertical: "top"},
			}
			locked := col.IsLocked(sec.Locked)
			styleTmpl := resolveStyle(sec.HeaderStyle, defaultHeader, locked)
			sid, err := s.exporter.createStyle(s.file, styleTmpl)
			if err != nil {
				return err
			}
			headers[i] = excelize.Cell{Value: col.Header, StyleID: sid}

			// Set Width?
			if col.Width > 0 {
				sw.SetColWidth(i+1, i+1, col.Width)
			}
		}

		if err := sw.SetRow(cell, headers); err != nil {
			return err
		}
		s.currentRow++
	}

	// 3. Data (if any static data)
	if sec.Data != nil {
		// Temporarily reuse Write logic by calling Write with NO validation check?
		// No, just copy paste core logic or refactor.
		// Refactoring `Write` to be usable for static data:
		// `Write` checks specific section ID.
		// Let's assume we can just call internal writeBatch(sec, data).
		return s.writeBatch(sw, sec, sec.Data)
	}

	return nil
}

func (s *Streamer) writeBatch(sw *excelize.StreamWriter, sec *SectionConfig, data interface{}) error {
	// Resolve Columns
	if len(sec.Columns) == 0 {
		sec.Columns = mergeColumns(data, sec.Columns)
	}

	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Ptr {
		dataVal = dataVal.Elem()
	}
	if dataVal.Kind() != reflect.Slice {
		// Single item?
		return nil
	}

	// Prepare styles
	colStyles := make([]int, len(sec.Columns))
	for j, col := range sec.Columns {
		locked := col.IsLocked(sec.Locked)
		var defaultDataStyle *StyleTemplate
		if sec.Type == SectionTypeHidden {
			defaultDataStyle = &StyleTemplate{Fill: &FillTemplate{Color: "FFFF00"}}
		}
		styleTmpl := resolveStyle(sec.DataStyle, defaultDataStyle, locked)
		sid, err := s.exporter.createStyle(s.file, styleTmpl)
		if err != nil {
			return err
		}
		colStyles[j] = sid
	}

	// Write rows
	for i := 0; i < dataVal.Len(); i++ {
		item := dataVal.Index(i)
		cell, _ := excelize.CoordinatesToCellName(1, s.currentRow)
		rowVals := make([]interface{}, len(sec.Columns))
		for j, col := range sec.Columns {
			val := s.exporter.extractValue(item, col.FieldName)
			if col.Formatter != nil {
				val = col.Formatter(val)
			} else if col.FormatterName != "" {
				if fmtFunc, ok := s.exporter.formatters[col.FormatterName]; ok {
					val = fmtFunc(val)
				}
			}
			rowVals[j] = excelize.Cell{
				Value:   val,
				StyleID: colStyles[j],
			}
		}
		if err := sw.SetRow(cell, rowVals); err != nil {
			return err
		}
		s.currentRow++
	}
	return nil
}
