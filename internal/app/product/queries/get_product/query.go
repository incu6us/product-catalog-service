package get_product

import (
	"context"

	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
)

// Reader defines the read interface needed by this query.
type Reader interface {
	GetByID(ctx context.Context, id string) (*ProductDTO, error)
}

// Query handles get product by ID.
type Query struct {
	readModel Reader
}

// NewQuery creates a new Query.
func NewQuery(readModel Reader) *Query {
	return &Query{readModel: readModel}
}

// Execute retrieves a product by ID.
func (q *Query) Execute(ctx context.Context, id string) (dto *ProductDTO, err error) {
	ctx, end := observability.StartSpan(ctx, "GetProduct")
	defer func() { end(err) }()

	return q.readModel.GetByID(ctx, id)
}
