package pgexcel

import (
	"context"
	"database/sql"
	"io"
)

// Exporter defines the interface for PostgreSQL to Excel export operations.
type Exporter interface {
	// Export executes the query and exports results to Excel
	Export(ctx context.Context, writer io.Writer, opts ...ExportOption) error

	// ExportToFile is a convenience method that exports to a file path
	ExportToFile(ctx context.Context, filepath string, opts ...ExportOption) error

	// AddSheet adds another sheet to the export
	AddSheet(query string, sheetName string, opts ...SheetOption) Exporter
}

// ProtectionRule defines how cells should be protected in the Excel sheet.
type ProtectionRule interface {
	// Apply applies the protection rule to the sheet
	Apply(sheetProtection *SheetProtection) error

	// Description returns a human-readable description of the rule
	Description() string
}

// CellRange represents a range of cells in Excel notation (e.g., "A1:B10")
type CellRange struct {
	StartCol string // e.g., "A"
	StartRow int    // 1-based
	EndCol   string // e.g., "B"
	EndRow   int    // 1-based
}

// String returns Excel notation for the range
func (r CellRange) String() string {
	return r.StartCol + string(rune(r.StartRow+'0')) + ":" + r.EndCol + string(rune(r.EndRow+'0'))
}

// ColumnRange represents one or more columns
type ColumnRange struct {
	Start string // e.g., "A"
	End   string // e.g., "C", or same as Start for single column
}

// RowRange represents one or more rows
type RowRange struct {
	Start int // 1-based
	End   int // 1-based
}

// SheetProtection holds the protection configuration for a sheet
type SheetProtection struct {
	Password       string
	ProtectSheet   bool
	LockedCells    map[string]bool // cell coordinate -> locked
	LockedRanges   []CellRange
	UnlockedRanges []CellRange
	LockedColumns  []ColumnRange
	LockedRows     []RowRange

	// Advanced protection options
	AllowFormatCells      bool
	AllowFormatColumns    bool
	AllowFormatRows       bool
	AllowInsertColumns    bool
	AllowInsertRows       bool
	AllowInsertHyperlinks bool
	AllowDeleteColumns    bool
	AllowDeleteRows       bool
	AllowSort             bool
	AllowFilter           bool
	AllowPivotTables      bool
}

// NewSheetProtection creates a new SheetProtection with sensible defaults
func NewSheetProtection() *SheetProtection {
	return &SheetProtection{
		ProtectSheet:          true,
		LockedCells:           make(map[string]bool),
		AllowFormatCells:      false,
		AllowFormatColumns:    false,
		AllowFormatRows:       false,
		AllowInsertColumns:    false,
		AllowInsertRows:       false,
		AllowInsertHyperlinks: false,
		AllowDeleteColumns:    false,
		AllowDeleteRows:       false,
		AllowSort:             false,
		AllowFilter:           true, // Usually allow filtering
		AllowPivotTables:      false,
	}
}

// ExportConfig holds configuration for an export operation
type ExportConfig struct {
	// Query configuration
	Query string
	Args  []interface{}

	// Sheet configuration
	SheetName string

	// Display options
	IncludeHeaders bool
	FreezeHeader   bool
	AutoFilter     bool
	AutoFitColumns bool
	MaxColumnWidth int

	// Protection
	Protection *SheetProtection

	// Styling
	HeaderStyle  *CellStyle
	DataStyles   map[string]*CellStyle // column name -> style
	DateFormat   string
	TimeFormat   string
	NumberFormat string

	// Multi-sheet support
	Sheets []SheetConfig
}

// SheetConfig holds configuration for a single sheet
type SheetConfig struct {
	Query      string
	Args       []interface{}
	SheetName  string
	Protection *SheetProtection
	Options    []SheetOption
}

// CellStyle defines styling for cells
type CellStyle struct {
	FontName   string
	FontSize   float64
	FontBold   bool
	FontItalic bool
	FontColor  string

	FillColor   string
	FillPattern int

	Alignment     string // "left", "center", "right"
	VerticalAlign string // "top", "middle", "bottom"

	BorderStyle string
	BorderColor string

	NumberFormat string

	WrapText bool
	Locked   bool
}

// DefaultHeaderStyle returns a default style for headers
func DefaultHeaderStyle() *CellStyle {
	return &CellStyle{
		FontName:      "Arial",
		FontSize:      11,
		FontBold:      true,
		FontColor:     "#FFFFFF",
		FillColor:     "#4472C4",
		FillPattern:   1,
		Alignment:     "center",
		VerticalAlign: "middle",
		Locked:        true,
	}
}

// DefaultDataStyle returns a default style for data cells
func DefaultDataStyle() *CellStyle {
	return &CellStyle{
		FontName:      "Arial",
		FontSize:      10,
		Alignment:     "left",
		VerticalAlign: "middle",
		Locked:        true,
	}
}

// ExportOption is a functional option for Export operations
type ExportOption func(*ExportConfig) error

// SheetOption is a functional option for sheet configuration
type SheetOption func(*SheetConfig) error

// RowFilterFunc is a function that determines if a row should be locked
// It receives the row number (1-based) and the row data
type RowFilterFunc func(rowNum int, rowData []interface{}) bool

// CellFilterFunc is a function that determines if a cell should be locked
// It receives the column name, row number, and cell value
type CellFilterFunc func(col string, rowNum int, value interface{}) bool

// DB is an interface that abstracts database operations
// This allows for easier testing and supports both *sql.DB and *sql.Tx
type DB interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}
