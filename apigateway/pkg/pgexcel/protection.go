package pgexcel

import (
	"fmt"
	"strings"
)

// Protection rules implementations

// lockAllExceptRule locks all cells except specified ranges
type lockAllExceptRule struct {
	exceptions []ProtectionRule
}

func (r *lockAllExceptRule) Apply(sp *SheetProtection) error {
	sp.ProtectSheet = true
	// First, lock everything by default (cells are locked by default in Excel)
	// Then apply exceptions to unlock specific parts
	for _, exception := range r.exceptions {
		if err := exception.Apply(sp); err != nil {
			return fmt.Errorf("applying exception: %w", err)
		}
	}
	return nil
}

func (r *lockAllExceptRule) Description() string {
	return "Lock all cells except specified exceptions"
}

// LockAllExcept creates a rule that locks all cells except the specified ones
func LockAllExcept(exceptions ...ProtectionRule) ProtectionRule {
	return &lockAllExceptRule{exceptions: exceptions}
}

// unlockColumnsRule unlocks specific columns
type unlockColumnsRule struct {
	columns []ColumnRange
}

func (r *unlockColumnsRule) Apply(sp *SheetProtection) error {
	// In Excel protection model, we mark these columns as unlocked
	// Implementation will need to set cell styles with Locked=false
	// We store this info for later application
	for _, col := range r.columns {
		sp.UnlockedRanges = append(sp.UnlockedRanges, CellRange{
			StartCol: col.Start,
			StartRow: 1,
			EndCol:   col.End,
			EndRow:   1048576, // Max Excel row
		})
	}
	return nil
}

func (r *unlockColumnsRule) Description() string {
	var cols []string
	for _, c := range r.columns {
		if c.Start == c.End {
			cols = append(cols, c.Start)
		} else {
			cols = append(cols, c.Start+"-"+c.End)
		}
	}
	return fmt.Sprintf("Unlock columns: %s", strings.Join(cols, ", "))
}

// Columns creates a ProtectionRule that unlocks the specified columns
func Columns(cols ...string) ProtectionRule {
	ranges := make([]ColumnRange, len(cols))
	for i, col := range cols {
		ranges[i] = ColumnRange{Start: col, End: col}
	}
	return &unlockColumnsRule{columns: ranges}
}

// ColumnRange creates a ProtectionRule that unlocks a range of columns
func ColumnRangeRule(start, end string) ProtectionRule {
	return &unlockColumnsRule{columns: []ColumnRange{{Start: start, End: end}}}
}

// lockColumnsRule locks specific columns
type lockColumnsRule struct {
	columns []ColumnRange
}

func (r *lockColumnsRule) Apply(sp *SheetProtection) error {
	sp.LockedColumns = append(sp.LockedColumns, r.columns...)
	return nil
}

func (r *lockColumnsRule) Description() string {
	var cols []string
	for _, c := range r.columns {
		if c.Start == c.End {
			cols = append(cols, c.Start)
		} else {
			cols = append(cols, c.Start+"-"+c.End)
		}
	}
	return fmt.Sprintf("Lock columns: %s", strings.Join(cols, ", "))
}

// LockColumns creates a ProtectionRule that locks the specified columns
func LockColumns(cols ...string) ProtectionRule {
	ranges := make([]ColumnRange, len(cols))
	for i, col := range cols {
		ranges[i] = ColumnRange{Start: col, End: col}
	}
	return &lockColumnsRule{columns: ranges}
}

// unlockRangeRule unlocks a specific cell range
type unlockRangeRule struct {
	ranges []CellRange
}

func (r *unlockRangeRule) Apply(sp *SheetProtection) error {
	sp.UnlockedRanges = append(sp.UnlockedRanges, r.ranges...)
	return nil
}

func (r *unlockRangeRule) Description() string {
	var rangeStrs []string
	for _, rng := range r.ranges {
		rangeStrs = append(rangeStrs, rng.String())
	}
	return fmt.Sprintf("Unlock ranges: %s", strings.Join(rangeStrs, ", "))
}

// UnlockRange creates a ProtectionRule that unlocks specific cell ranges
// Accepts Excel notation like "A1:B10"
func UnlockRange(ranges ...string) ProtectionRule {
	cellRanges := make([]CellRange, len(ranges))
	for i, r := range ranges {
		cellRanges[i] = parseCellRange(r)
	}
	return &unlockRangeRule{ranges: cellRanges}
}

// lockRangeRule locks a specific cell range
type lockRangeRule struct {
	ranges []CellRange
}

func (r *lockRangeRule) Apply(sp *SheetProtection) error {
	sp.LockedRanges = append(sp.LockedRanges, r.ranges...)
	return nil
}

func (r *lockRangeRule) Description() string {
	var rangeStrs []string
	for _, rng := range r.ranges {
		rangeStrs = append(rangeStrs, rng.String())
	}
	return fmt.Sprintf("Lock ranges: %s", strings.Join(rangeStrs, ", "))
}

// LockRanges creates a ProtectionRule that locks specific cell ranges
func LockRanges(ranges ...string) ProtectionRule {
	cellRanges := make([]CellRange, len(ranges))
	for i, r := range ranges {
		cellRanges[i] = parseCellRange(r)
	}
	return &lockRangeRule{ranges: cellRanges}
}

// lockRowsRule locks specific rows
type lockRowsRule struct {
	rows []RowRange
}

func (r *lockRowsRule) Apply(sp *SheetProtection) error {
	sp.LockedRows = append(sp.LockedRows, r.rows...)
	return nil
}

func (r *lockRowsRule) Description() string {
	var rowStrs []string
	for _, row := range r.rows {
		if row.Start == row.End {
			rowStrs = append(rowStrs, fmt.Sprintf("%d", row.Start))
		} else {
			rowStrs = append(rowStrs, fmt.Sprintf("%d-%d", row.Start, row.End))
		}
	}
	return fmt.Sprintf("Lock rows: %s", strings.Join(rowStrs, ", "))
}

// LockRows creates a ProtectionRule that locks specific rows
func LockRows(rows ...int) ProtectionRule {
	ranges := make([]RowRange, len(rows))
	for i, row := range rows {
		ranges[i] = RowRange{Start: row, End: row}
	}
	return &lockRowsRule{rows: ranges}
}

// LockRowsAbove locks all rows above (and including) the specified row
func LockRowsAbove(row int) ProtectionRule {
	return &lockRowsRule{rows: []RowRange{{Start: 1, End: row}}}
}

// LockRowsBelow locks all rows below (and including) the specified row
func LockRowsBelow(row int) ProtectionRule {
	return &lockRowsRule{rows: []RowRange{{Start: row, End: 1048576}}}
}

// conditionalRowLockRule locks rows based on a condition
type conditionalRowLockRule struct {
	filter RowFilterFunc
}

func (r *conditionalRowLockRule) Apply(sp *SheetProtection) error {
	// This will be applied during the actual data writing phase
	// We can't apply it now because we don't have the data yet
	// Store it for later application
	return nil
}

func (r *conditionalRowLockRule) Description() string {
	return "Lock rows conditionally based on data"
}

// LockRowsWhere creates a ProtectionRule that locks rows matching a condition
func LockRowsWhere(filter RowFilterFunc) ProtectionRule {
	return &conditionalRowLockRule{filter: filter}
}

// conditionalCellLockRule locks cells based on a condition
type conditionalCellLockRule struct {
	filter CellFilterFunc
}

func (r *conditionalCellLockRule) Apply(sp *SheetProtection) error {
	// This will be applied during the actual data writing phase
	return nil
}

func (r *conditionalCellLockRule) Description() string {
	return "Lock cells conditionally based on data"
}

// LockCellsWhere creates a ProtectionRule that locks cells matching a condition
func LockCellsWhere(filter CellFilterFunc) ProtectionRule {
	return &conditionalCellLockRule{filter: filter}
}

// compositeProtectionRule combines multiple protection rules
type compositeProtectionRule struct {
	rules []ProtectionRule
}

func (r *compositeProtectionRule) Apply(sp *SheetProtection) error {
	for _, rule := range r.rules {
		if err := rule.Apply(sp); err != nil {
			return fmt.Errorf("applying composite rule: %w", err)
		}
	}
	return nil
}

func (r *compositeProtectionRule) Description() string {
	var descriptions []string
	for _, rule := range r.rules {
		descriptions = append(descriptions, rule.Description())
	}
	return strings.Join(descriptions, "; ")
}

// CombineRules combines multiple protection rules into one
func CombineRules(rules ...ProtectionRule) ProtectionRule {
	return &compositeProtectionRule{rules: rules}
}

// Helper function to parse Excel cell range notation
func parseCellRange(rangeStr string) CellRange {
	// Simple parser for ranges like "A1:B10"
	parts := strings.Split(rangeStr, ":")
	if len(parts) != 2 {
		return CellRange{}
	}

	start := parseCellRef(parts[0])
	end := parseCellRef(parts[1])

	return CellRange{
		StartCol: start.col,
		StartRow: start.row,
		EndCol:   end.col,
		EndRow:   end.row,
	}
}

type cellRef struct {
	col string
	row int
}

// Simple cell reference parser (e.g., "A1" -> col: "A", row: 1)
func parseCellRef(ref string) cellRef {
	var col string
	var row int

	for i, ch := range ref {
		if ch >= 'A' && ch <= 'Z' {
			col += string(ch)
		} else if ch >= 'a' && ch <= 'z' {
			col += string(ch - 32) // Convert to uppercase
		} else if ch >= '0' && ch <= '9' {
			// Parse remaining as number
			fmt.Sscanf(ref[i:], "%d", &row)
			break
		}
	}

	return cellRef{col: col, row: row}
}
