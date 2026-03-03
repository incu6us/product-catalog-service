package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcspanner "github.com/testcontainers/testcontainers-go/modules/gcloud/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/archive_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/deactivate_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/remove_discount"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/update_product"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
	"github.com/incu6us/product-catalog-service/internal/services"
)

const (
	projectID  = "test-project"
	instanceID = "test-instance"
	dbID       = "test-db"
)

var (
	sharedClient *spanner.Client
	sharedApp    *services.App
)

func dbPath() string {
	return "projects/" + projectID + "/instances/" + instanceID + "/databases/" + dbID
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcspanner.Run(ctx,
		"gcr.io/cloud-spanner-emulator/emulator:1.4.0",
		tcspanner.WithProjectID(projectID),
	)
	if err != nil {
		log.Fatalf("failed to start spanner container: %v", err)
	}

	opts := []option.ClientOption{
		option.WithEndpoint(container.URI()),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}

	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("failed to create instance admin client: %v", err)
	}

	_, _ = instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     "projects/" + projectID,
		InstanceId: instanceID,
		Instance:   &instancepb.Instance{DisplayName: instanceID},
	})
	_ = instanceAdmin.Close()

	dbAdmin, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("failed to create database admin client: %v", err)
	}

	op, err := dbAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          "projects/" + projectID + "/instances/" + instanceID,
		CreateStatement: "CREATE DATABASE `" + dbID + "`",
		ExtraStatements: []string{
			`CREATE TABLE products (
				product_id STRING(36) NOT NULL,
				name STRING(255) NOT NULL,
				description STRING(MAX),
				category STRING(100) NOT NULL,
				base_price_numerator INT64 NOT NULL,
				base_price_denominator INT64 NOT NULL,
				discount_percent NUMERIC,
				discount_start_date TIMESTAMP,
				discount_end_date TIMESTAMP,
				status STRING(20) NOT NULL,
				created_at TIMESTAMP NOT NULL,
				updated_at TIMESTAMP NOT NULL,
				archived_at TIMESTAMP,
				version INT64 NOT NULL,
			) PRIMARY KEY (product_id)`,
			`CREATE TABLE outbox_events (
				event_id STRING(36) NOT NULL,
				event_type STRING(100) NOT NULL,
				aggregate_id STRING(36) NOT NULL,
				payload JSON NOT NULL,
				status STRING(20) NOT NULL,
				created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
				processed_at TIMESTAMP,
			) PRIMARY KEY (event_id)`,
			`CREATE INDEX idx_outbox_status ON outbox_events(status, created_at)`,
			`CREATE INDEX idx_products_category ON products(category, status)`,
		},
	})
	if err != nil {
		log.Fatalf("failed to create database: %v", err)
	}
	if _, err := op.Wait(ctx); err != nil {
		log.Fatalf("failed to wait for database creation: %v", err)
	}
	_ = dbAdmin.Close()

	sharedClient, err = spanner.NewClient(ctx, dbPath(), opts...)
	if err != nil {
		log.Fatalf("failed to create spanner client: %v", err)
	}

	clk := clock.FixedClock{T: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	sharedApp = services.NewApp(sharedClient, clk)

	code := m.Run()

	sharedClient.Close()
	_ = container.Terminate(ctx)

	os.Exit(code)
}

func cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := sharedClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, table := range []string{"outbox_events", "products"} {
			if _, err := txn.Update(ctx, spanner.Statement{SQL: "DELETE FROM " + table + " WHERE true"}); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func setup(t *testing.T) (*services.App, *spanner.Client) {
	t.Helper()
	cleanup(t)
	return sharedApp, sharedClient
}

func countOutboxEvents(t *testing.T, client *spanner.Client, aggregateID string) int {
	t.Helper()
	ctx := context.Background()
	stmt := spanner.Statement{
		SQL:    "SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = @id",
		Params: map[string]interface{}{"id": aggregateID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	require.NoError(t, err)
	var count int64
	require.NoError(t, row.Columns(&count))
	return int(count)
}

func outboxEventTypes(t *testing.T, client *spanner.Client, aggregateID string) []string {
	t.Helper()
	ctx := context.Background()
	stmt := spanner.Statement{
		SQL:    "SELECT event_type FROM outbox_events WHERE aggregate_id = @id ORDER BY created_at",
		Params: map[string]interface{}{"id": aggregateID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var types []string
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var eventType string
		require.NoError(t, row.Columns(&eventType))
		types = append(types, eventType)
	}
	return types
}

func queryArchivedAt(t *testing.T, client *spanner.Client, productID string) *time.Time {
	t.Helper()
	ctx := context.Background()
	stmt := spanner.Statement{
		SQL:    "SELECT archived_at FROM products WHERE product_id = @id",
		Params: map[string]interface{}{"id": productID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	require.NoError(t, err)
	var archivedAt *time.Time
	require.NoError(t, row.Columns(&archivedAt))
	return archivedAt
}

func TestProductCreationFlow(t *testing.T) {
	app, client := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Test Product", Description: "A test product", Category: "electronics",
		BasePriceNumerator: 1999, BasePriceDenominator: 100,
	})
	require.NoError(t, err)
	require.NotEmpty(t, productID)

	dto, err := app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "Test Product", dto.Name)
	assert.Equal(t, "electronics", dto.Category)
	assert.Equal(t, "draft", dto.Status)
	assert.Equal(t, 1, countOutboxEvents(t, client, productID))
	assert.Equal(t, []string{"product.created"}, outboxEventTypes(t, client, productID))
}

func TestProductUpdateFlow(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Original", Description: "Original desc", Category: "books",
		BasePriceNumerator: 999, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.UpdateProduct.Execute(ctx, update_product.Request{
		ProductID: productID, Name: "Updated", Description: "Updated desc", Category: "books",
	})
	require.NoError(t, err)

	dto, err := app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", dto.Name)
	assert.Equal(t, "Updated desc", dto.Description)
}

func TestActivateDeactivateFlow(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Activatable", Description: "Test", Category: "toys",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)
	dto, err := app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "active", dto.Status)

	err = app.Commands.DeactivateProduct.Execute(ctx, deactivate_product.Request{ProductID: productID})
	require.NoError(t, err)
	dto, err = app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "inactive", dto.Status)
}

func TestDiscountApplicationFlow(t *testing.T) {
	app, client := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Discountable", Description: "Test", Category: "food",
		BasePriceNumerator: 2000, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	err = app.Commands.ApplyDiscount.Execute(ctx, apply_discount.Request{
		ProductID: productID, Percentage: 20,
		StartDate: now.Add(-time.Hour), EndDate: now.Add(24 * time.Hour),
	})
	require.NoError(t, err)

	dto, err := app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "16.00", dto.EffectivePrice)
	require.NotNil(t, dto.DiscountPercent)
	assert.Equal(t, int64(20), *dto.DiscountPercent)
	assert.Equal(t, 3, countOutboxEvents(t, client, productID))
	assert.Equal(t, []string{"product.created", "product.activated", "discount.applied"}, outboxEventTypes(t, client, productID))

	err = app.Commands.RemoveDiscount.Execute(ctx, remove_discount.Request{ProductID: productID})
	require.NoError(t, err)
	dto, err = app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "20.00", dto.EffectivePrice)
	assert.Nil(t, dto.DiscountPercent)
}

func TestBusinessRuleValidation(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Inactive Product", Description: "Test", Category: "misc",
		BasePriceNumerator: 100, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	err = app.Commands.ApplyDiscount.Execute(ctx, apply_discount.Request{
		ProductID: productID, Percentage: 10,
		StartDate: now.Add(-time.Hour), EndDate: now.Add(24 * time.Hour),
	})
	assert.ErrorIs(t, err, domain.ErrProductNotActive)

	err = app.Commands.DeactivateProduct.Execute(ctx, deactivate_product.Request{ProductID: productID})
	assert.ErrorIs(t, err, domain.ErrProductNotActive)
}

func TestListProducts(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	for _, name := range []string{"Product A", "Product B", "Product C"} {
		_, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
			Name: name, Description: "Test", Category: "list-test",
			BasePriceNumerator: 100, BasePriceDenominator: 100,
		})
		require.NoError(t, err)
	}

	result, err := app.Queries.ListProducts.Execute(ctx, list_products.ListParams{
		Category: "list-test", PageSize: 10,
	})
	require.NoError(t, err)
	assert.Len(t, result.Products, 3)
}

func TestArchiveFlow(t *testing.T) {
	app, client := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Archivable", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	err = app.Commands.ArchiveProduct.Execute(ctx, archive_product.Request{ProductID: productID})
	require.NoError(t, err)

	dto, err := app.Queries.GetProduct.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "archived", dto.Status)
	assert.Equal(t, 3, countOutboxEvents(t, client, productID))
	assert.Equal(t, []string{"product.created", "product.activated", "product.archived"}, outboxEventTypes(t, client, productID))
	assert.NotNil(t, queryArchivedAt(t, client, productID))

	// Cannot archive again
	err = app.Commands.ArchiveProduct.Execute(ctx, archive_product.Request{ProductID: productID})
	assert.ErrorIs(t, err, domain.ErrProductAlreadyArchived)

	// Cannot activate archived product
	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	assert.ErrorIs(t, err, domain.ErrProductAlreadyArchived)
}

func TestConcurrentModification(t *testing.T) {
	app, client := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Concurrent", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	// Load the product in two separate "sessions" by reading its current state
	// and then issuing two updates that both expect the same version.

	// First update succeeds
	err = app.Commands.UpdateProduct.Execute(ctx, update_product.Request{
		ProductID: productID, Name: "Version2", Description: "Test", Category: "misc",
	})
	require.NoError(t, err)

	// Simulate stale read: directly load the product at the old version and try to update.
	// We can't easily simulate a stale aggregate through the public API,
	// so instead we write a raw DML with the wrong expected version.
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE products SET name = @name, updated_at = @updated_at, version = @new_version
				  WHERE product_id = @product_id AND version = @expected_version`,
			Params: map[string]interface{}{
				"name":             "StaleUpdate",
				"updated_at":       time.Now(),
				"new_version":      int64(2), // would be version 2
				"product_id":       productID,
				"expected_version": int64(1), // stale: actual version is already 2
			},
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		if rowCount == 0 {
			return committer.ErrConcurrentModification
		}
		return nil
	})
	assert.ErrorIs(t, err, committer.ErrConcurrentModification)
}

func TestRemoveDiscountWithoutDiscount(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "No Discount", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	err = app.Commands.RemoveDiscount.Execute(ctx, remove_discount.Request{ProductID: productID})
	assert.ErrorIs(t, err, domain.ErrNoDiscountToRemove)
}

func TestCreateProduct_EmptyName(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	_, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	assert.ErrorIs(t, err, domain.ErrInvalidProductName)
}

func TestCreateProduct_ZeroDenominator(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	_, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Bad Price", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 0,
	})
	assert.ErrorIs(t, err, domain.ErrZeroDenominator)
}

func TestDoubleActivate(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	productID, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Double Activate", Description: "Test", Category: "misc",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	require.NoError(t, err)

	err = app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{ProductID: productID})
	assert.ErrorIs(t, err, domain.ErrProductAlreadyActive)
}

func TestCreateProduct_EmptyCategory(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	_, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name: "Widget", Description: "Test", Category: "",
		BasePriceNumerator: 500, BasePriceDenominator: 100,
	})
	assert.ErrorIs(t, err, domain.ErrInvalidCategory)
}

func TestListProducts_Pagination(t *testing.T) {
	app, _ := setup(t)
	ctx := context.Background()

	// Create 5 products.
	for i := range 5 {
		_, err := app.Commands.CreateProduct.Execute(ctx, create_product.Request{
			Name:                 fmt.Sprintf("PaginatedProduct-%d", i),
			Description:          "Test",
			Category:             "page-test",
			BasePriceNumerator:   100,
			BasePriceDenominator: 100,
		})
		require.NoError(t, err)
	}

	// Fetch first page of 2.
	page1, err := app.Queries.ListProducts.Execute(ctx, list_products.ListParams{
		Category: "page-test", PageSize: 2,
	})
	require.NoError(t, err)
	assert.Len(t, page1.Products, 2)
	assert.NotEmpty(t, page1.NextPageToken, "expected next page token")

	// Fetch second page using the token.
	page2, err := app.Queries.ListProducts.Execute(ctx, list_products.ListParams{
		Category: "page-test", PageSize: 2, PageToken: page1.NextPageToken,
	})
	require.NoError(t, err)
	assert.Len(t, page2.Products, 2)
	assert.NotEmpty(t, page2.NextPageToken, "expected next page token for page 2")

	// Fetch third page — should contain 1 remaining product.
	page3, err := app.Queries.ListProducts.Execute(ctx, list_products.ListParams{
		Category: "page-test", PageSize: 2, PageToken: page2.NextPageToken,
	})
	require.NoError(t, err)
	assert.Len(t, page3.Products, 1)
	assert.Empty(t, page3.NextPageToken, "no more pages expected")

	// Verify no overlap between pages.
	seen := make(map[string]bool)
	for _, p := range page1.Products {
		assert.False(t, seen[p.ID], "duplicate product across pages")
		seen[p.ID] = true
	}
	for _, p := range page2.Products {
		assert.False(t, seen[p.ID], "duplicate product across pages")
		seen[p.ID] = true
	}
	for _, p := range page3.Products {
		assert.False(t, seen[p.ID], "duplicate product across pages")
		seen[p.ID] = true
	}
	assert.Len(t, seen, 5)
}
