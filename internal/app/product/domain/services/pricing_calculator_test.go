package services_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/app/product/domain/services"
)

var now = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func TestCalculateEffectivePrice_NoDiscount(t *testing.T) {
	calc := services.NewPricingCalculator()
	price, _ := domain.NewMoney(2000, 100)

	result := calc.CalculateEffectivePrice(price, nil, now)
	assert.Equal(t, "20.00", result.String())
}

func TestCalculateEffectivePrice_ActiveDiscount(t *testing.T) {
	calc := services.NewPricingCalculator()
	price, _ := domain.NewMoney(2000, 100)

	discount := domain.ReconstructDiscount(
		new(big.Rat).SetInt64(20),
		now.Add(-time.Hour),
		now.Add(24*time.Hour),
	)

	result := calc.CalculateEffectivePrice(price, discount, now)
	assert.Equal(t, "16.00", result.String())
}

func TestCalculateEffectivePrice_ExpiredDiscount(t *testing.T) {
	calc := services.NewPricingCalculator()
	price, _ := domain.NewMoney(2000, 100)

	discount := domain.ReconstructDiscount(
		new(big.Rat).SetInt64(20),
		now.Add(-48*time.Hour),
		now.Add(-24*time.Hour),
	)

	result := calc.CalculateEffectivePrice(price, discount, now)
	assert.Equal(t, "20.00", result.String())
}

func TestCalculateEffectivePrice_FutureDiscount(t *testing.T) {
	calc := services.NewPricingCalculator()
	price, _ := domain.NewMoney(2000, 100)

	discount := domain.ReconstructDiscount(
		new(big.Rat).SetInt64(20),
		now.Add(24*time.Hour),
		now.Add(48*time.Hour),
	)

	result := calc.CalculateEffectivePrice(price, discount, now)
	assert.Equal(t, "20.00", result.String())
}

func TestCalculateEffectivePrice_50Percent(t *testing.T) {
	calc := services.NewPricingCalculator()
	price, _ := domain.NewMoney(1000, 100)

	discount := domain.ReconstructDiscount(
		new(big.Rat).SetInt64(50),
		now.Add(-time.Hour),
		now.Add(time.Hour),
	)

	result := calc.CalculateEffectivePrice(price, discount, now)
	require.Equal(t, "5.00", result.String())
}
