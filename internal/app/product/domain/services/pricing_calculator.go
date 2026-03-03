package services

import (
	"math/big"
	"time"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
)

// PricingCalculator is a domain service for calculating effective prices.
type PricingCalculator struct{}

// NewPricingCalculator creates a new PricingCalculator.
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculateEffectivePrice returns the effective price after applying a discount.
// If the discount is nil or not currently active, the base price is returned.
func (pc *PricingCalculator) CalculateEffectivePrice(basePrice *domain.Money, discount *domain.Discount, now time.Time) *domain.Money {
	if discount == nil || !discount.IsActive(now) {
		return basePrice
	}

	// discount percentage is 0-100, convert to fraction
	discountFraction := new(big.Rat).Quo(discount.Percentage(), new(big.Rat).SetInt64(100))
	discountMultiplier := domain.MoneyFromRat(discountFraction)
	discountAmount := basePrice.Mul(discountMultiplier)

	return basePrice.Sub(discountAmount)
}
