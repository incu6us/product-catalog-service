package list_products

import "time"

// ListParams defines parameters for listing products.
type ListParams struct {
	Category string
	Status   string
	PageSize int
	PageToken string
}

// ProductItem is a summary of a product in a list.
type ProductItem struct {
	ID             string
	Name           string
	Category       string
	BasePrice      string
	EffectivePrice string
	Status         string
	CreatedAt      time.Time
}

// ListResult contains the paginated list of products.
type ListResult struct {
	Products      []*ProductItem
	NextPageToken string
}
