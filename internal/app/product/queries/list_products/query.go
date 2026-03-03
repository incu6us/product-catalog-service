package list_products

import (
	"context"

	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
)

// Reader defines the read interface needed by this query.
type Reader interface {
	List(ctx context.Context, params ListParams) (*ListResult, error)
}

// Query handles listing products with filters and pagination.
type Query struct {
	readModel Reader
}

// NewQuery creates a new Query.
func NewQuery(readModel Reader) *Query {
	return &Query{readModel: readModel}
}

// Execute retrieves a paginated list of products.
func (q *Query) Execute(ctx context.Context, params ListParams) (result *ListResult, err error) {
	ctx, end := observability.StartSpan(ctx, "ListProducts")
	defer func() { end(err) }()

	return q.readModel.List(ctx, params)
}
