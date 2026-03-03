package create_product

import (
	"context"

	"github.com/google/uuid"

	"github.com/incu6us/product-catalog-service/internal/app/product/contracts"
	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
)

// Request contains parameters for creating a product.
type Request struct {
	Name                 string
	Description          string
	Category             string
	BasePriceNumerator   int64
	BasePriceDenominator int64
}

// Interactor handles the create product usecase.
type Interactor struct {
	repo      contracts.ProductRepository
	outbox    contracts.OutboxRepository
	executor  *committer.Executor
	clock     clock.Clock
}

// NewInteractor creates a new Interactor.
func NewInteractor(
	repo contracts.ProductRepository,
	outbox contracts.OutboxRepository,
	executor *committer.Executor,
	clk clock.Clock,
) *Interactor {
	return &Interactor{
		repo:     repo,
		outbox:   outbox,
		executor: executor,
		clock:    clk,
	}
}

// Execute creates a new product.
func (it *Interactor) Execute(ctx context.Context, req Request) (id string, err error) {
	ctx, end := observability.StartSpan(ctx, "CreateProduct")
	defer func() { end(err) }()

	basePrice, err := domain.NewMoney(req.BasePriceNumerator, req.BasePriceDenominator)
	if err != nil {
		return "", err
	}

	product, err := domain.NewProduct(
		uuid.New().String(),
		req.Name,
		req.Description,
		req.Category,
		basePrice,
		it.clock.Now(),
	)
	if err != nil {
		return "", err
	}

	plan := committer.NewPlan()

	insertMut, err := it.repo.InsertMut(product)
	if err != nil {
		return "", err
	}
	if err := plan.AddMutation(insertMut); err != nil {
		return "", err
	}

	outboxMuts, err := it.outbox.InsertMuts(product.DomainEvents(), product.ID(), product.Version())
	if err != nil {
		return "", err
	}
	if err := plan.AddMutations(outboxMuts); err != nil {
		return "", err
	}

	if err := it.executor.Apply(ctx, plan); err != nil {
		return "", err
	}

	return product.ID(), nil
}
