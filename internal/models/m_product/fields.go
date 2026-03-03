package m_product

// Table and column name constants for the products table.
const (
	TableName = "products"

	ColProductID            = "product_id"
	ColName                 = "name"
	ColDescription          = "description"
	ColCategory             = "category"
	ColBasePriceNumerator   = "base_price_numerator"
	ColBasePriceDenominator = "base_price_denominator"
	ColDiscountPercent      = "discount_percent"
	ColDiscountStartDate    = "discount_start_date"
	ColDiscountEndDate      = "discount_end_date"
	ColStatus               = "status"
	ColCreatedAt            = "created_at"
	ColUpdatedAt            = "updated_at"
	ColArchivedAt           = "archived_at"
	ColVersion              = "version"
)

// AllColumns returns all columns for SELECT queries.
func AllColumns() []string {
	return []string{
		ColProductID,
		ColName,
		ColDescription,
		ColCategory,
		ColBasePriceNumerator,
		ColBasePriceDenominator,
		ColDiscountPercent,
		ColDiscountStartDate,
		ColDiscountEndDate,
		ColStatus,
		ColCreatedAt,
		ColUpdatedAt,
		ColArchivedAt,
		ColVersion,
	}
}
