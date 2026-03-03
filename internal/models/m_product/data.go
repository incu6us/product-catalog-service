package m_product

import (
	"math/big"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents a row in the products table.
type Data struct {
	ProductID            string
	Name                 string
	Description          string
	Category             string
	BasePriceNumerator   int64
	BasePriceDenominator int64
	DiscountPercent      *big.Rat
	DiscountStartDate    *time.Time
	DiscountEndDate      *time.Time
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ArchivedAt           *time.Time
	Version              int64
}

// InsertMut creates a Spanner insert mutation for this product.
func (d *Data) InsertMut() *spanner.Mutation {
	cols := AllColumns()
	vals := []interface{}{
		d.ProductID,
		d.Name,
		d.Description,
		d.Category,
		d.BasePriceNumerator,
		d.BasePriceDenominator,
		d.DiscountPercent,
		d.DiscountStartDate,
		d.DiscountEndDate,
		d.Status,
		d.CreatedAt,
		d.UpdatedAt,
		d.ArchivedAt,
		d.Version,
	}
	return spanner.Insert(TableName, cols, vals)
}

// UpdateMut creates a Spanner update mutation for the given columns.
func UpdateMut(productID string, updates map[string]interface{}) *spanner.Mutation {
	cols := []string{ColProductID}
	vals := []interface{}{productID}

	for col, val := range updates {
		cols = append(cols, col)
		vals = append(vals, val)
	}

	return spanner.Update(TableName, cols, vals)
}

// ReadStatement creates a Spanner read statement for fetching a product by ID.
func ReadStatement(productID string) spanner.Statement {
	return spanner.Statement{
		SQL:    "SELECT " + JoinColumns() + " FROM " + TableName + " WHERE " + ColProductID + " = @id",
		Params: map[string]interface{}{"id": productID},
	}
}

// JoinColumns returns all column names joined by commas for SQL SELECT statements.
func JoinColumns() string {
	return strings.Join(AllColumns(), ", ")
}
