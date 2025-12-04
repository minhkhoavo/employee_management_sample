package pgexcel

// StyleBuilder provides a fluent API for building cell styles
type StyleBuilder struct {
	style *CellStyle
}

// NewStyleBuilder creates a new style builder with default values
func NewStyleBuilder() *StyleBuilder {
	return &StyleBuilder{
		style: &CellStyle{
			FontName:      "Arial",
			FontSize:      10,
			Alignment:     "left",
			VerticalAlign: "middle",
			Locked:        true,
		},
	}
}

// Font sets the font properties
func (b *StyleBuilder) Font(name string, size float64) *StyleBuilder {
	b.style.FontName = name
	b.style.FontSize = size
	return b
}

// Bold sets the font to bold
func (b *StyleBuilder) Bold() *StyleBuilder {
	b.style.FontBold = true
	return b
}

// Italic sets the font to italic
func (b *StyleBuilder) Italic() *StyleBuilder {
	b.style.FontItalic = true
	return b
}

// FontColor sets the font color (hex format)
func (b *StyleBuilder) FontColor(color string) *StyleBuilder {
	b.style.FontColor = color
	return b
}

// Fill sets the cell background color
func (b *StyleBuilder) Fill(color string) *StyleBuilder {
	b.style.FillColor = color
	b.style.FillPattern = 1
	return b
}

// Align sets the horizontal alignment
func (b *StyleBuilder) Align(alignment string) *StyleBuilder {
	b.style.Alignment = alignment
	return b
}

// VAlign sets the vertical alignment
func (b *StyleBuilder) VAlign(alignment string) *StyleBuilder {
	b.style.VerticalAlign = alignment
	return b
}

// Border sets the border style
func (b *StyleBuilder) Border(style, color string) *StyleBuilder {
	b.style.BorderStyle = style
	b.style.BorderColor = color
	return b
}

// NumberFormat sets the number format
func (b *StyleBuilder) NumberFormat(format string) *StyleBuilder {
	b.style.NumberFormat = format
	return b
}

// WrapText enables text wrapping
func (b *StyleBuilder) WrapText() *StyleBuilder {
	b.style.WrapText = true
	return b
}

// Locked sets the cell locked state
func (b *StyleBuilder) Locked(locked bool) *StyleBuilder {
	b.style.Locked = locked
	return b
}

// Build returns the built style
func (b *StyleBuilder) Build() *CellStyle {
	return b.style
}

// Pre-defined styles

// HeaderStyleBlue returns a blue header style
func HeaderStyleBlue() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 11).
		Bold().
		FontColor("#FFFFFF").
		Fill("#4472C4").
		Align("center").
		VAlign("middle").
		Locked(true).
		Build()
}

// HeaderStyleGreen returns a green header style
func HeaderStyleGreen() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 11).
		Bold().
		FontColor("#FFFFFF").
		Fill("#70AD47").
		Align("center").
		VAlign("middle").
		Locked(true).
		Build()
}

// HeaderStyleDark returns a dark header style
func HeaderStyleDark() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 11).
		Bold().
		FontColor("#FFFFFF").
		Fill("#44546A").
		Align("center").
		VAlign("middle").
		Locked(true).
		Build()
}

// DataStyleEditable returns a style for editable data cells
func DataStyleEditable() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 10).
		Fill("#FFF2CC").
		Locked(false).
		Build()
}

// DataStyleReadOnly returns a style for read-only data cells
func DataStyleReadOnly() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 10).
		Fill("#F2F2F2").
		Locked(true).
		Build()
}

// DataStyleHighlight returns a highlighted style for important data
func DataStyleHighlight() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 10).
		Fill("#FFE699").
		Bold().
		Locked(true).
		Build()
}

// DateStyle returns a style for date cells
func DateStyle(format string) *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 10).
		NumberFormat(format).
		Align("center").
		Locked(true).
		Build()
}

// CurrencyStyle returns a style for currency cells
func CurrencyStyle(symbol string) *CellStyle {
	format := symbol + "#,##0.00"
	return NewStyleBuilder().
		Font("Arial", 10).
		NumberFormat(format).
		Align("right").
		Locked(true).
		Build()
}

// PercentageStyle returns a style for percentage cells
func PercentageStyle() *CellStyle {
	return NewStyleBuilder().
		Font("Arial", 10).
		NumberFormat("0.00%").
		Align("right").
		Locked(true).
		Build()
}
