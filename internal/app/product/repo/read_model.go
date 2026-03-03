package repo

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/app/product/domain/services"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/incu6us/product-catalog-service/internal/models/m_product"
	"github.com/incu6us/product-catalog-service/internal/pkg/clock"
)

const maxPageSize = 100

// ReadModel implements the product read model using Spanner.
type ReadModel struct {
	client     *spanner.Client
	clock      clock.Clock
	calculator *services.PricingCalculator
}

// NewReadModel creates a new ReadModel.
func NewReadModel(client *spanner.Client, clk clock.Clock) *ReadModel {
	return &ReadModel{
		client:     client,
		clock:      clk,
		calculator: services.NewPricingCalculator(),
	}
}

type productRow struct {
	productID            string
	name                 string
	description          string
	category             string
	basePriceNumerator   int64
	basePriceDenominator int64
	discountPercent      *big.Rat
	discountStartDate    *time.Time
	discountEndDate      *time.Time
	status               string
	createdAt            time.Time
	updatedAt            time.Time
	archivedAt           *time.Time
	version              int64
}

func scanProductRow(row *spanner.Row) (*productRow, error) {
	var r productRow
	if err := row.Columns(
		&r.productID,
		&r.name,
		&r.description,
		&r.category,
		&r.basePriceNumerator,
		&r.basePriceDenominator,
		&r.discountPercent,
		&r.discountStartDate,
		&r.discountEndDate,
		&r.status,
		&r.createdAt,
		&r.updatedAt,
		&r.archivedAt,
		&r.version,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetByID returns a product DTO by its ID.
func (rm *ReadModel) GetByID(ctx context.Context, id string) (*get_product.ProductDTO, error) {
	stmt := m_product.ReadStatement(id)
	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}

	r, err := scanProductRow(row)
	if err != nil {
		return nil, err
	}

	basePrice, err := domain.NewMoney(r.basePriceNumerator, r.basePriceDenominator)
	if err != nil {
		return nil, err
	}

	var discount *domain.Discount
	if r.discountPercent != nil && r.discountStartDate != nil && r.discountEndDate != nil {
		discount = domain.ReconstructDiscount(r.discountPercent, *r.discountStartDate, *r.discountEndDate)
	}

	now := rm.clock.Now()
	effectivePrice := rm.calculator.CalculateEffectivePrice(basePrice, discount, now)

	dto := &get_product.ProductDTO{
		ID:             r.productID,
		Name:           r.name,
		Description:    r.description,
		Category:       r.category,
		BasePrice:      basePrice.String(),
		EffectivePrice: effectivePrice.String(),
		Status:         r.status,
		CreatedAt:      r.createdAt,
		UpdatedAt:      r.updatedAt,
	}

	if r.discountPercent != nil {
		if !r.discountPercent.IsInt() {
			return nil, fmt.Errorf("discount percentage is not an integer: %s", r.discountPercent.RatString())
		}
		num := r.discountPercent.Num()
		if !num.IsInt64() {
			return nil, fmt.Errorf("discount percentage overflows int64: %s", num.String())
		}
		v := num.Int64()
		dto.DiscountPercent = &v
	}

	return dto, nil
}

// List returns a paginated list of products.
func (rm *ReadModel) List(ctx context.Context, params list_products.ListParams) (*list_products.ListResult, error) {
	query := "SELECT " + m_product.JoinColumns() + " FROM " + m_product.TableName + " WHERE 1=1"
	qParams := map[string]interface{}{}

	if params.Category != "" {
		query += " AND " + m_product.ColCategory + " = @category"
		qParams["category"] = params.Category
	}
	if params.Status != "" {
		query += " AND " + m_product.ColStatus + " = @status"
		qParams["status"] = params.Status
	}
	if params.PageToken != "" {
		query += " AND (" + m_product.ColCreatedAt + " < " +
			"(SELECT " + m_product.ColCreatedAt + " FROM " + m_product.TableName + " WHERE " + m_product.ColProductID + " = @page_token)" +
			" OR (" + m_product.ColCreatedAt + " = " +
			"(SELECT " + m_product.ColCreatedAt + " FROM " + m_product.TableName + " WHERE " + m_product.ColProductID + " = @page_token)" +
			" AND " + m_product.ColProductID + " < @page_token))"
		qParams["page_token"] = params.PageToken
	}

	query += " ORDER BY " + m_product.ColCreatedAt + " DESC, " + m_product.ColProductID + " DESC"

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	query += " LIMIT @limit"
	qParams["limit"] = int64(pageSize + 1) // fetch one extra to determine next page

	stmt := spanner.Statement{SQL: query, Params: qParams}
	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var items []*list_products.ProductItem
	now := rm.clock.Now()

	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, err
		}

		r, err := scanProductRow(row)
		if err != nil {
			return nil, err
		}

		basePrice, err := domain.NewMoney(r.basePriceNumerator, r.basePriceDenominator)
		if err != nil {
			return nil, err
		}

		var discount *domain.Discount
		if r.discountPercent != nil && r.discountStartDate != nil && r.discountEndDate != nil {
			discount = domain.ReconstructDiscount(r.discountPercent, *r.discountStartDate, *r.discountEndDate)
		}

		effectivePrice := rm.calculator.CalculateEffectivePrice(basePrice, discount, now)

		items = append(items, &list_products.ProductItem{
			ID:             r.productID,
			Name:           r.name,
			Category:       r.category,
			BasePrice:      basePrice.String(),
			EffectivePrice: effectivePrice.String(),
			Status:         r.status,
			CreatedAt:      r.createdAt,
		})
	}

	result := &list_products.ListResult{}

	if len(items) > pageSize {
		result.Products = items[:pageSize]
		result.NextPageToken = items[pageSize-1].ID
	} else {
		result.Products = items
	}

	return result, nil
}
