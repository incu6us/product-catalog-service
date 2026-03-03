package contracts

import (
	"context"

	"github.com/incu6us/commitplan"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
)

// ProductRepository defines the interface for product persistence.
type ProductRepository interface {
	InsertMut(product *domain.Product) (*commitplan.Mutation, error)
	UpdateDML(product *domain.Product) *committer.DMLStatement
	FindByID(ctx context.Context, id string) (*domain.Product, error)
}
