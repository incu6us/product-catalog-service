package remove_discount

import (
	"context"

	"github.com/incu6us/product-catalog-service/internal/app/product/contracts"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
)

// Request contains parameters for removing a discount.
type Request struct {
	ProductID string
}

// Interactor handles the remove discount usecase.
type Interactor struct {
	repo     contracts.ProductRepository
	outbox   contracts.OutboxRepository
	executor *committer.Executor
	clock    clock.Clock
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

// Execute removes a discount from a product.
func (it *Interactor) Execute(ctx context.Context, req Request) (err error) {
	ctx, end := observability.StartSpan(ctx, "RemoveDiscount")
	defer func() { end(err) }()

	product, err := it.repo.FindByID(ctx, req.ProductID)
	if err != nil {
		return err
	}

	now := it.clock.Now()
	if err := product.RemoveDiscount(now); err != nil {
		return err
	}

	plan := committer.NewPlan()

	if dml := it.repo.UpdateDML(product); dml != nil {
		plan.AddDML(*dml)
	}

	outboxMuts, err := it.outbox.InsertMuts(product.DomainEvents(), product.ID(), product.Version())
	if err != nil {
		return err
	}
	if err := plan.AddMutations(outboxMuts); err != nil {
		return err
	}

	if plan.IsEmpty() {
		return nil
	}

	return it.executor.Apply(ctx, plan)
}
