package apply_discount

import (
	"context"
	"time"

	"github.com/incu6us/product-catalog-service/internal/app/product/contracts"
	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
	"github.com/incu6us/product-catalog-service/internal/pkg/observability"
)

// Request contains parameters for applying a discount.
type Request struct {
	ProductID  string
	Percentage int64
	StartDate  time.Time
	EndDate    time.Time
}

// Interactor handles the apply discount usecase.
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

// Execute applies a discount to a product.
func (it *Interactor) Execute(ctx context.Context, req Request) (err error) {
	ctx, end := observability.StartSpan(ctx, "ApplyDiscount")
	defer func() { end(err) }()

	product, err := it.repo.FindByID(ctx, req.ProductID)
	if err != nil {
		return err
	}

	discount, err := domain.NewDiscount(req.Percentage, req.StartDate, req.EndDate)
	if err != nil {
		return err
	}

	now := it.clock.Now()
	if err := product.ApplyDiscount(discount, now); err != nil {
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
