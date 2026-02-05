package domain

// ProductExcel represents a product with extensive image and metadata fields
// for Excel export demonstration with large datasets (200,000+ records)
type ProductExcel struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Image URLs
	ThumbnailImage string `json:"thumbnail_image"`
	PrimaryImage   string `json:"primary_image"`
	SecondaryImage string `json:"secondary_image"`
	DetailImage1   string `json:"detail_image_1"`
	DetailImage2   string `json:"detail_image_2"`
	DetailImage3   string `json:"detail_image_3"`
	DetailImage4   string `json:"detail_image_4"`
	DetailImage5   string `json:"detail_image_5"`

	// Metadata with 13 fields
	Metadata map[string]string `json:"metadata"`
}

// MetadataKeys defines the 13 required metadata fields
var MetadataKeys = []string{
	"brand",
	"category",
	"subcategory",
	"color",
	"size",
	"material",
	"weight",
	"country_of_origin",
	"manufacturer",
	"sku",
	"barcode",
	"warranty_period",
	"release_date",
}

// GetExcelHeaders returns the header row for Excel export
// Total columns: 2 (ID, Name) + 8 (Images) + 13 (Metadata) = 23 columns
func GetExcelHeaders() []string {
	headers := []string{
		"ID",
		"Name",
		"ThumbnailImage",
		"PrimaryImage",
		"SecondaryImage",
		"DetailImage1",
		"DetailImage2",
		"DetailImage3",
		"DetailImage4",
		"DetailImage5",
	}

	// Add metadata headers
	for _, key := range MetadataKeys {
		headers = append(headers, "Meta_"+key)
	}

	return headers
}

// ToExcelRow converts a ProductExcel to a row of interface{} for streaming
func (p *ProductExcel) ToExcelRow() []interface{} {
	row := make([]interface{}, 0, 23)

	// Basic fields
	row = append(row, p.ID)
	row = append(row, p.Name)

	// Image fields
	row = append(row, p.ThumbnailImage)
	row = append(row, p.PrimaryImage)
	row = append(row, p.SecondaryImage)
	row = append(row, p.DetailImage1)
	row = append(row, p.DetailImage2)
	row = append(row, p.DetailImage3)
	row = append(row, p.DetailImage4)
	row = append(row, p.DetailImage5)

	// Metadata fields (in order of MetadataKeys)
	for _, key := range MetadataKeys {
		if val, ok := p.Metadata[key]; ok {
			row = append(row, val)
		} else {
			row = append(row, "")
		}
	}

	return row
}
