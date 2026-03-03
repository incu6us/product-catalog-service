package contracts

import (
	"context"

	"github.com/incu6us/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
)

// ProductReadModel defines the interface for product read operations.
type ProductReadModel interface {
	GetByID(ctx context.Context, id string) (*get_product.ProductDTO, error)
	List(ctx context.Context, params list_products.ListParams) (*list_products.ListResult, error)
}
