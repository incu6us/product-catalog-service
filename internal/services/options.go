package services

import (
	"cloud.google.com/go/spanner"

	"github.com/incu6us/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/incu6us/product-catalog-service/internal/app/product/repo"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/archive_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/deactivate_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/remove_discount"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/update_product"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
)

// Commands groups all command interactors.
type Commands struct {
	CreateProduct     *create_product.Interactor
	UpdateProduct     *update_product.Interactor
	ApplyDiscount     *apply_discount.Interactor
	RemoveDiscount    *remove_discount.Interactor
	ActivateProduct   *activate_product.Interactor
	DeactivateProduct *deactivate_product.Interactor
	ArchiveProduct    *archive_product.Interactor
}

// Queries groups all query handlers.
type Queries struct {
	GetProduct   *get_product.Query
	ListProducts *list_products.Query
}

// App holds all application dependencies.
type App struct {
	Commands *Commands
	Queries  *Queries
}

// NewApp creates a fully wired App.
func NewApp(client *spanner.Client, clk clock.Clock) *App {
	executor := committer.NewExecutor(client)
	productRepo := repo.NewProductRepo(client)
	outboxRepo := repo.NewOutboxRepo()
	readModel := repo.NewReadModel(client, clk)

	commands := &Commands{
		CreateProduct:     create_product.NewInteractor(productRepo, outboxRepo, executor, clk),
		UpdateProduct:     update_product.NewInteractor(productRepo, outboxRepo, executor, clk),
		ApplyDiscount:     apply_discount.NewInteractor(productRepo, outboxRepo, executor, clk),
		RemoveDiscount:    remove_discount.NewInteractor(productRepo, outboxRepo, executor, clk),
		ActivateProduct:   activate_product.NewInteractor(productRepo, outboxRepo, executor, clk),
		DeactivateProduct: deactivate_product.NewInteractor(productRepo, outboxRepo, executor, clk),
		ArchiveProduct:    archive_product.NewInteractor(productRepo, outboxRepo, executor, clk),
	}

	queries := &Queries{
		GetProduct:   get_product.NewQuery(readModel),
		ListProducts: list_products.NewQuery(readModel),
	}

	return &App{
		Commands: commands,
		Queries:  queries,
	}
}
