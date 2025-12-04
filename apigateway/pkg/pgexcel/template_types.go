package pgexcel

// template_types.go - YAML-mappable types for report template configuration
// This provides an XSLT-like capability for defining Excel report layouts via YAML

// ReportTemplate represents the complete YAML template configuration
type ReportTemplate struct {
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Defaults    *TemplateDefaults `yaml:"defaults,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Sheets      []SheetTemplate   `yaml:"sheets"`
}

// TemplateDefaults holds default configurations applied to all sheets
type TemplateDefaults struct {
	HeaderStyle *StyleTemplate `yaml:"header_style,omitempty"`
	DataStyle   *StyleTemplate `yaml:"data_style,omitempty"`
	DateFormat  string         `yaml:"date_format,omitempty"`
	TimeFormat  string         `yaml:"time_format,omitempty"`
	NumFormat   string         `yaml:"number_format,omitempty"`
}

// SheetTemplate represents a single sheet configuration
type SheetTemplate struct {
	Name       string              `yaml:"name"`
	Query      string              `yaml:"query,omitempty"`
	QueryFile  string              `yaml:"query_file,omitempty"` // Load SQL from external file
	QueryArgs  []string            `yaml:"query_args,omitempty"` // Variable references for query params
	Columns    []ColumnTemplate    `yaml:"columns,omitempty"`
	Protection *ProtectionTemplate `yaml:"protection,omitempty"`
	Style      *SheetStyleTemplate `yaml:"style,omitempty"`
	Layout     *LayoutTemplate     `yaml:"layout,omitempty"`
}

// ColumnTemplate defines column-specific configurations
type ColumnTemplate struct {
	Name        string            `yaml:"name"`                  // DB column name (required)
	Header      string            `yaml:"header,omitempty"`      // Display header (defaults to Name)
	Width       float64           `yaml:"width,omitempty"`       // Column width
	Format      string            `yaml:"format,omitempty"`      // Number/date format
	Style       *StyleTemplate    `yaml:"style,omitempty"`       // Column-specific style
	Hidden      bool              `yaml:"hidden,omitempty"`      // Hide this column
	Formula     string            `yaml:"formula,omitempty"`     // Excel formula for calculated columns
	Conditional []ConditionalRule `yaml:"conditional,omitempty"` // Conditional formatting rules
}

// ConditionalRule defines conditional formatting based on cell values
type ConditionalRule struct {
	Condition string         `yaml:"condition"` // Expression: "> 100", "== 'ACTIVE'", "contains 'error'"
	Style     *StyleTemplate `yaml:"style"`     // Style to apply when condition is true
}

// ProtectionTemplate defines sheet protection configuration
type ProtectionTemplate struct {
	Password              string   `yaml:"password,omitempty"`
	LockSheet             bool     `yaml:"lock_sheet"`
	LockedColumns         []string `yaml:"locked_columns,omitempty"`   // Columns to lock (e.g., ["A", "B"])
	UnlockedColumns       []string `yaml:"unlocked_columns,omitempty"` // Columns to unlock
	LockedRows            []string `yaml:"locked_rows,omitempty"`      // Row ranges: "1", "1-5", "header"
	UnlockedRanges        []string `yaml:"unlocked_ranges,omitempty"`  // Cell ranges: "D2:E1000"
	AllowFilter           bool     `yaml:"allow_filter,omitempty"`
	AllowSort             bool     `yaml:"allow_sort,omitempty"`
	AllowFormatCells      bool     `yaml:"allow_format_cells,omitempty"`
	AllowFormatColumns    bool     `yaml:"allow_format_columns,omitempty"`
	AllowFormatRows       bool     `yaml:"allow_format_rows,omitempty"`
	AllowInsertRows       bool     `yaml:"allow_insert_rows,omitempty"`
	AllowInsertColumns    bool     `yaml:"allow_insert_columns,omitempty"`
	AllowInsertHyperlinks bool     `yaml:"allow_insert_hyperlinks,omitempty"`
	AllowDeleteRows       bool     `yaml:"allow_delete_rows,omitempty"`
	AllowDeleteColumns    bool     `yaml:"allow_delete_columns,omitempty"`
	AllowPivotTables      bool     `yaml:"allow_pivot_tables,omitempty"`
}

// SheetStyleTemplate defines sheet-level style overrides
type SheetStyleTemplate struct {
	HeaderStyle *StyleTemplate `yaml:"header_style,omitempty"`
	DataStyle   *StyleTemplate `yaml:"data_style,omitempty"`
}

// LayoutTemplate controls sheet layout options
type LayoutTemplate struct {
	FreezeRows      int    `yaml:"freeze_rows,omitempty"`
	FreezeCols      int    `yaml:"freeze_cols,omitempty"`
	AutoFilter      bool   `yaml:"auto_filter,omitempty"`
	AutoFitCols     bool   `yaml:"auto_fit_columns,omitempty"`
	MaxColWidth     int    `yaml:"max_column_width,omitempty"`
	ShowGridlines   *bool  `yaml:"show_gridlines,omitempty"`   // Pointer to distinguish unset from false
	PrintArea       string `yaml:"print_area,omitempty"`       // e.g., "A1:G100"
	PageOrientation string `yaml:"page_orientation,omitempty"` // "portrait" or "landscape"
}

// StyleTemplate for cell/column/header styling
type StyleTemplate struct {
	Font         *FontTemplate   `yaml:"font,omitempty"`
	Fill         *FillTemplate   `yaml:"fill,omitempty"`
	Border       *BorderTemplate `yaml:"border,omitempty"`
	Alignment    string          `yaml:"alignment,omitempty"`  // "left", "center", "right"
	VAlignment   string          `yaml:"valignment,omitempty"` // "top", "middle", "bottom"
	NumberFormat string          `yaml:"number_format,omitempty"`
	WrapText     bool            `yaml:"wrap_text,omitempty"`
	Locked       *bool           `yaml:"locked,omitempty"` // Pointer to distinguish unset
}

// FontTemplate defines font properties
type FontTemplate struct {
	Name   string  `yaml:"name,omitempty"`
	Size   float64 `yaml:"size,omitempty"`
	Bold   bool    `yaml:"bold,omitempty"`
	Italic bool    `yaml:"italic,omitempty"`
	Color  string  `yaml:"color,omitempty"` // Hex color e.g., "#FF0000"
}

// FillTemplate defines cell fill/background
type FillTemplate struct {
	Color   string `yaml:"color,omitempty"`   // Hex color e.g., "#FFFF00"
	Pattern int    `yaml:"pattern,omitempty"` // Fill pattern (1 = solid)
}

// BorderTemplate defines cell borders
type BorderTemplate struct {
	Style string `yaml:"style,omitempty"` // "thin", "medium", "thick"
	Color string `yaml:"color,omitempty"` // Hex color
}

// Helper methods for ColumnTemplate

// GetHeader returns the display header (falls back to column name)
func (c *ColumnTemplate) GetHeader() string {
	if c.Header != "" {
		return c.Header
	}
	return c.Name
}

// Helper to check if a style has any values set
func (s *StyleTemplate) IsEmpty() bool {
	if s == nil {
		return true
	}
	return s.Font == nil && s.Fill == nil && s.Border == nil &&
		s.Alignment == "" && s.VAlignment == "" && s.NumberFormat == "" &&
		!s.WrapText && s.Locked == nil
}

// ToCellStyle converts StyleTemplate to the existing CellStyle type
func (s *StyleTemplate) ToCellStyle() *CellStyle {
	if s == nil || s.IsEmpty() {
		return nil
	}

	style := &CellStyle{
		Alignment:     s.Alignment,
		VerticalAlign: s.VAlignment,
		NumberFormat:  s.NumberFormat,
		WrapText:      s.WrapText,
	}

	if s.Font != nil {
		style.FontName = s.Font.Name
		style.FontSize = s.Font.Size
		style.FontBold = s.Font.Bold
		style.FontItalic = s.Font.Italic
		style.FontColor = s.Font.Color
	}

	if s.Fill != nil {
		style.FillColor = s.Fill.Color
		style.FillPattern = s.Fill.Pattern
		if style.FillPattern == 0 && style.FillColor != "" {
			style.FillPattern = 1 // Default to solid fill
		}
	}

	if s.Border != nil {
		style.BorderStyle = s.Border.Style
		style.BorderColor = s.Border.Color
	}

	if s.Locked != nil {
		style.Locked = *s.Locked
	}

	return style
}

// Merge merges another StyleTemplate into this one (other takes precedence)
func (s *StyleTemplate) Merge(other *StyleTemplate) *StyleTemplate {
	if other == nil {
		return s
	}
	if s == nil {
		return other
	}

	result := &StyleTemplate{
		Alignment:    s.Alignment,
		VAlignment:   s.VAlignment,
		NumberFormat: s.NumberFormat,
		WrapText:     s.WrapText,
		Locked:       s.Locked,
	}

	// Merge font
	if s.Font != nil || other.Font != nil {
		result.Font = &FontTemplate{}
		if s.Font != nil {
			*result.Font = *s.Font
		}
		if other.Font != nil {
			if other.Font.Name != "" {
				result.Font.Name = other.Font.Name
			}
			if other.Font.Size != 0 {
				result.Font.Size = other.Font.Size
			}
			if other.Font.Bold {
				result.Font.Bold = true
			}
			if other.Font.Italic {
				result.Font.Italic = true
			}
			if other.Font.Color != "" {
				result.Font.Color = other.Font.Color
			}
		}
	}

	// Merge fill
	if s.Fill != nil || other.Fill != nil {
		result.Fill = &FillTemplate{}
		if s.Fill != nil {
			*result.Fill = *s.Fill
		}
		if other.Fill != nil {
			if other.Fill.Color != "" {
				result.Fill.Color = other.Fill.Color
			}
			if other.Fill.Pattern != 0 {
				result.Fill.Pattern = other.Fill.Pattern
			}
		}
	}

	// Merge border
	if s.Border != nil || other.Border != nil {
		result.Border = &BorderTemplate{}
		if s.Border != nil {
			*result.Border = *s.Border
		}
		if other.Border != nil {
			if other.Border.Style != "" {
				result.Border.Style = other.Border.Style
			}
			if other.Border.Color != "" {
				result.Border.Color = other.Border.Color
			}
		}
	}

	// Override scalar values from other
	if other.Alignment != "" {
		result.Alignment = other.Alignment
	}
	if other.VAlignment != "" {
		result.VAlignment = other.VAlignment
	}
	if other.NumberFormat != "" {
		result.NumberFormat = other.NumberFormat
	}
	if other.WrapText {
		result.WrapText = true
	}
	if other.Locked != nil {
		result.Locked = other.Locked
	}

	return result
}
