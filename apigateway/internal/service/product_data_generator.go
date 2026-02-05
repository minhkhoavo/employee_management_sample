package service

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
)

// ProductDataGenerator generates seed data for ProductExcel
type ProductDataGenerator struct {
	rng *rand.Rand
}

// NewProductDataGenerator creates a new generator with a fixed seed for reproducibility
func NewProductDataGenerator(seed int64) *ProductDataGenerator {
	return &ProductDataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Sample data pools for generating realistic product data
var (
	brands = []string{
		"Nike", "Adidas", "Puma", "Reebok", "New Balance",
		"Under Armour", "Asics", "Converse", "Vans", "Fila",
		"Samsung", "Apple", "Sony", "LG", "Panasonic",
		"Dell", "HP", "Lenovo", "Asus", "Acer",
	}

	categories = []string{
		"Electronics", "Clothing", "Footwear", "Accessories",
		"Sports", "Home & Garden", "Beauty", "Toys",
		"Books", "Food & Beverages",
	}

	subcategories = map[string][]string{
		"Electronics":      {"Smartphones", "Laptops", "Tablets", "Cameras", "Audio"},
		"Clothing":         {"Shirts", "Pants", "Dresses", "Jackets", "Sweaters"},
		"Footwear":         {"Sneakers", "Boots", "Sandals", "Loafers", "Running"},
		"Accessories":      {"Bags", "Watches", "Jewelry", "Belts", "Wallets"},
		"Sports":           {"Fitness", "Outdoor", "Team Sports", "Water Sports", "Winter Sports"},
		"Home & Garden":    {"Furniture", "Decor", "Kitchen", "Garden", "Lighting"},
		"Beauty":           {"Skincare", "Makeup", "Hair Care", "Fragrance", "Nail Care"},
		"Toys":             {"Action Figures", "Board Games", "Puzzles", "Dolls", "Educational"},
		"Books":            {"Fiction", "Non-Fiction", "Comics", "Textbooks", "Children"},
		"Food & Beverages": {"Snacks", "Beverages", "Organic", "Frozen", "Canned"},
	}

	colors = []string{
		"Red", "Blue", "Green", "Black", "White",
		"Yellow", "Orange", "Purple", "Pink", "Gray",
		"Navy", "Teal", "Maroon", "Olive", "Coral",
	}

	sizes = []string{
		"XS", "S", "M", "L", "XL", "XXL",
		"One Size", "Small", "Medium", "Large",
		"6", "7", "8", "9", "10", "11", "12",
	}

	materials = []string{
		"Cotton", "Polyester", "Leather", "Denim", "Wool",
		"Silk", "Linen", "Nylon", "Velvet", "Canvas",
		"Aluminum", "Plastic", "Glass", "Wood", "Stainless Steel",
	}

	countries = []string{
		"USA", "China", "Japan", "Germany", "South Korea",
		"Vietnam", "India", "Italy", "France", "UK",
		"Thailand", "Indonesia", "Mexico", "Brazil", "Canada",
	}

	manufacturers = []string{
		"Foxconn", "Samsung Electronics", "LG Electronics", "Sony Corp", "Apple Inc",
		"Nike Inc", "Adidas AG", "PVH Corp", "VF Corporation", "Hanesbrands",
		"Global Manufacturing", "Asia Pacific Industrial", "European Goods Ltd",
		"Premium Products Inc", "Quality Makers Co",
	}

	productAdjectives = []string{
		"Premium", "Classic", "Pro", "Ultra", "Max",
		"Sport", "Casual", "Elegant", "Modern", "Vintage",
		"Limited Edition", "Signature", "Essential", "Performance", "Comfort",
	}

	productNouns = []string{
		"Runner", "Walker", "Trainer", "Player", "Master",
		"Elite", "Champion", "Winner", "Star", "Prime",
		"One", "Plus", "Air", "Tech", "Flex",
	}
)

// GenerateProduct generates a single product with index for easy identification
func (g *ProductDataGenerator) GenerateProduct(index int) *domain.ProductExcel {
	brand := brands[g.rng.Intn(len(brands))]
	category := categories[g.rng.Intn(len(categories))]
	subCats := subcategories[category]
	subcategory := subCats[g.rng.Intn(len(subCats))]
	color := colors[g.rng.Intn(len(colors))]
	size := sizes[g.rng.Intn(len(sizes))]
	material := materials[g.rng.Intn(len(materials))]
	country := countries[g.rng.Intn(len(countries))]
	manufacturer := manufacturers[g.rng.Intn(len(manufacturers))]

	adjective := productAdjectives[g.rng.Intn(len(productAdjectives))]
	noun := productNouns[g.rng.Intn(len(productNouns))]

	// Generate unique ID with index for easy tracking
	productID := fmt.Sprintf("PROD-%06d-%04d", index, g.rng.Intn(10000))

	// Generate product name that's easy to identify
	productName := fmt.Sprintf("%s %s %s %d", brand, adjective, noun, index)

	// Generate image URLs with index for traceability
	baseURL := "https://cdn.example.com/products"

	// Generate SKU prefix safely (handle short brand names)
	brandPrefix := brand
	if len(brandPrefix) > 3 {
		brandPrefix = brandPrefix[:3]
	}

	// Generate metadata with 13 fields
	metadata := map[string]string{
		"brand":             brand,
		"category":          category,
		"subcategory":       subcategory,
		"color":             color,
		"size":              size,
		"material":          material,
		"weight":            fmt.Sprintf("%.2f kg", g.rng.Float64()*10+0.1),
		"country_of_origin": country,
		"manufacturer":      manufacturer,
		"sku":               fmt.Sprintf("SKU-%s-%06d", brandPrefix, index),
		"barcode":           fmt.Sprintf("%013d", 1000000000000+int64(index)),
		"warranty_period":   fmt.Sprintf("%d months", (g.rng.Intn(4)+1)*6),
		"release_date":      generateRandomDate(g.rng),
	}

	return &domain.ProductExcel{
		ID:             productID,
		Name:           productName,
		ThumbnailImage: fmt.Sprintf("%s/%d/thumb_%d.jpg", baseURL, index, index),
		PrimaryImage:   fmt.Sprintf("%s/%d/primary_%d.jpg", baseURL, index, index),
		SecondaryImage: fmt.Sprintf("%s/%d/secondary_%d.jpg", baseURL, index, index),
		DetailImage1:   fmt.Sprintf("%s/%d/detail1_%d.jpg", baseURL, index, index),
		DetailImage2:   fmt.Sprintf("%s/%d/detail2_%d.jpg", baseURL, index, index),
		DetailImage3:   fmt.Sprintf("%s/%d/detail3_%d.jpg", baseURL, index, index),
		DetailImage4:   fmt.Sprintf("%s/%d/detail4_%d.jpg", baseURL, index, index),
		DetailImage5:   fmt.Sprintf("%s/%d/detail5_%d.jpg", baseURL, index, index),
		Metadata:       metadata,
	}
}

// GenerateBatch generates a batch of products for memory-efficient processing
// Returns a channel that yields products one at a time
func (g *ProductDataGenerator) GenerateBatch(startIndex, count int) <-chan *domain.ProductExcel {
	ch := make(chan *domain.ProductExcel, 100) // Buffer for better throughput

	go func() {
		defer close(ch)
		for i := 0; i < count; i++ {
			ch <- g.GenerateProduct(startIndex + i)
		}
	}()

	return ch
}

// GenerateAll generates all products with a callback for each product
// This avoids storing all 200,000 products in memory at once
func (g *ProductDataGenerator) GenerateAll(totalCount int, callback func(product *domain.ProductExcel) error) error {
	for i := 0; i < totalCount; i++ {
		product := g.GenerateProduct(i)
		if err := callback(product); err != nil {
			return err
		}
	}
	return nil
}

// generateRandomDate generates a random date in the past 5 years
func generateRandomDate(rng *rand.Rand) string {
	now := time.Now()
	daysAgo := rng.Intn(365 * 5) // Up to 5 years ago
	date := now.AddDate(0, 0, -daysAgo)
	return date.Format("2006-01-02")
}
