package handler

// ProductDetailResponse response chính
type ProductDetailResponse struct {
	Item    ProductItemDTO     `json:"item"`
	Details []ProductDetailDTO `json:"details"`
}

// ProductItemDTO chứa info cơ bản
type ProductItemDTO struct {
	ID    int64  `json:"id"`
	Brand string `json:"brand"`
}

// ProductDetailDTO chứa merged info
type ProductDetailDTO struct {
	ID        int64  `json:"id"`
	Brand     string `json:"brand"`
	Country   string `json:"country"`
	Place     string `json:"place"`
	Year      int    `json:"year"`
	SubNumber int    `json:"sub_number"`
	Content   string `json:"content"`
}
