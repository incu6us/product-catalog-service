package get_product

import "time"

// ProductDTO is the read model for a single product.
type ProductDTO struct {
	ID             string
	Name           string
	Description    string
	Category       string
	BasePrice      string
	EffectivePrice string
	DiscountPercent *int64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
