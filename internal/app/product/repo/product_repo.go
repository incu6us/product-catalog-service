package repo

import (
	"context"
	"math/big"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/incu6us/commitplan"
	"google.golang.org/api/iterator"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/models/m_product"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
)

// ProductRepo is the Spanner implementation of contracts.ProductRepository.
type ProductRepo struct {
	client *spanner.Client
}

// NewProductRepo creates a new ProductRepo.
func NewProductRepo(client *spanner.Client) *ProductRepo {
	return &ProductRepo{client: client}
}

// InsertMut creates a commitplan insert mutation for the product.
func (r *ProductRepo) InsertMut(product *domain.Product) (*commitplan.Mutation, error) {
	num, err := product.BasePrice().SafeNumerator()
	if err != nil {
		return nil, err
	}
	denom, err := product.BasePrice().SafeDenominator()
	if err != nil {
		return nil, err
	}

	data := &m_product.Data{
		ProductID:            product.ID(),
		Name:                 product.Name(),
		Description:          product.Description(),
		Category:             product.Category(),
		BasePriceNumerator:   num,
		BasePriceDenominator: denom,
		Status:               string(product.Status()),
		CreatedAt:            product.CreatedAt(),
		UpdatedAt:            product.UpdatedAt(),
		Version:              product.Version(),
	}

	if d := product.Discount(); d != nil {
		data.DiscountPercent = d.Percentage()
		start := d.StartDate()
		end := d.EndDate()
		data.DiscountStartDate = &start
		data.DiscountEndDate = &end
	}

	return &commitplan.Mutation{SpannerMut: data.InsertMut()}, nil
}

// UpdateDML creates a DML statement for updating a product with optimistic locking.
func (r *ProductRepo) UpdateDML(product *domain.Product) *committer.DMLStatement {
	if !product.Changes().HasChanges() {
		return nil
	}

	changes := product.Changes()
	setClauses := []string{}
	params := map[string]interface{}{
		"product_id":       product.ID(),
		"expected_version": product.Version() - 1,
		"new_version":      product.Version(),
	}

	if changes.Dirty(domain.FieldName) {
		setClauses = append(setClauses, m_product.ColName+" = @name")
		params["name"] = product.Name()
	}
	if changes.Dirty(domain.FieldDescription) {
		setClauses = append(setClauses, m_product.ColDescription+" = @description")
		params["description"] = product.Description()
	}
	if changes.Dirty(domain.FieldCategory) {
		setClauses = append(setClauses, m_product.ColCategory+" = @category")
		params["category"] = product.Category()
	}
	if changes.Dirty(domain.FieldDiscount) {
		if d := product.Discount(); d != nil {
			setClauses = append(setClauses, m_product.ColDiscountPercent+" = @discount_percent")
			params["discount_percent"] = d.Percentage()
			start := d.StartDate()
			end := d.EndDate()
			setClauses = append(setClauses, m_product.ColDiscountStartDate+" = @discount_start_date")
			params["discount_start_date"] = start
			setClauses = append(setClauses, m_product.ColDiscountEndDate+" = @discount_end_date")
			params["discount_end_date"] = end
		} else {
			setClauses = append(setClauses, m_product.ColDiscountPercent+" = @discount_percent")
			params["discount_percent"] = (*big.Rat)(nil)
			setClauses = append(setClauses, m_product.ColDiscountStartDate+" = @discount_start_date")
			params["discount_start_date"] = (*time.Time)(nil)
			setClauses = append(setClauses, m_product.ColDiscountEndDate+" = @discount_end_date")
			params["discount_end_date"] = (*time.Time)(nil)
		}
	}
	if changes.Dirty(domain.FieldStatus) {
		setClauses = append(setClauses, m_product.ColStatus+" = @status")
		params["status"] = string(product.Status())
	}
	if changes.Dirty(domain.FieldArchivedAt) {
		setClauses = append(setClauses, m_product.ColArchivedAt+" = @archived_at")
		params["archived_at"] = product.ArchivedAt()
	}

	setClauses = append(setClauses, m_product.ColUpdatedAt+" = @updated_at")
	params["updated_at"] = product.UpdatedAt()

	setClauses = append(setClauses, m_product.ColVersion+" = @new_version")

	sql := "UPDATE " + m_product.TableName + " SET " + strings.Join(setClauses, ", ") +
		" WHERE " + m_product.ColProductID + " = @product_id" +
		" AND " + m_product.ColVersion + " = @expected_version"

	return &committer.DMLStatement{SQL: sql, Params: params}
}

// FindByID loads a product from Spanner.
func (r *ProductRepo) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	stmt := m_product.ReadStatement(id)

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}

	var data m_product.Data
	if err := row.Columns(
		&data.ProductID,
		&data.Name,
		&data.Description,
		&data.Category,
		&data.BasePriceNumerator,
		&data.BasePriceDenominator,
		&data.DiscountPercent,
		&data.DiscountStartDate,
		&data.DiscountEndDate,
		&data.Status,
		&data.CreatedAt,
		&data.UpdatedAt,
		&data.ArchivedAt,
		&data.Version,
	); err != nil {
		return nil, err
	}

	basePrice, err := domain.NewMoney(data.BasePriceNumerator, data.BasePriceDenominator)
	if err != nil {
		return nil, err
	}

	var discount *domain.Discount
	if data.DiscountPercent != nil && data.DiscountStartDate != nil && data.DiscountEndDate != nil {
		discount = domain.ReconstructDiscount(data.DiscountPercent, *data.DiscountStartDate, *data.DiscountEndDate)
	}

	return domain.ReconstructProduct(
		data.ProductID,
		data.Name,
		data.Description,
		data.Category,
		basePrice,
		discount,
		domain.ProductStatus(data.Status),
		data.Version,
		data.CreatedAt,
		data.UpdatedAt,
		data.ArchivedAt,
	), nil
}
