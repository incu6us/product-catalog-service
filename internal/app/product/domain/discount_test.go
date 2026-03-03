package domain_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
)

func TestNewDiscount_Valid(t *testing.T) {
	start := now.Add(-time.Hour)
	end := now.Add(24 * time.Hour)
	d, err := domain.NewDiscount(20, start, end)
	require.NoError(t, err)
	assert.Equal(t, start, d.StartDate())
	assert.Equal(t, end, d.EndDate())
}

func TestNewDiscount_ZeroPercentage(t *testing.T) {
	_, err := domain.NewDiscount(0, now, now.Add(time.Hour))
	assert.ErrorIs(t, err, domain.ErrInvalidDiscountPercentage)
}

func TestNewDiscount_NegativePercentage(t *testing.T) {
	_, err := domain.NewDiscount(-5, now, now.Add(time.Hour))
	assert.ErrorIs(t, err, domain.ErrInvalidDiscountPercentage)
}

func TestNewDiscount_Over100(t *testing.T) {
	_, err := domain.NewDiscount(101, now, now.Add(time.Hour))
	assert.ErrorIs(t, err, domain.ErrInvalidDiscountPercentage)
}

func TestNewDiscount_100Percent(t *testing.T) {
	d, err := domain.NewDiscount(100, now, now.Add(time.Hour))
	require.NoError(t, err)
	pct, err := d.SafePercentageInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(100), pct)
}

func TestNewDiscount_InvalidPeriod(t *testing.T) {
	_, err := domain.NewDiscount(20, now, now)
	assert.ErrorIs(t, err, domain.ErrInvalidDiscountPeriod)

	_, err = domain.NewDiscount(20, now.Add(time.Hour), now)
	assert.ErrorIs(t, err, domain.ErrInvalidDiscountPeriod)
}

func TestDiscount_IsValidAt(t *testing.T) {
	start := now
	end := now.Add(24 * time.Hour)
	d, _ := domain.NewDiscount(10, start, end)

	assert.True(t, d.IsValidAt(start))
	assert.True(t, d.IsValidAt(end))
	assert.True(t, d.IsValidAt(now.Add(12*time.Hour)))
	assert.False(t, d.IsValidAt(now.Add(-time.Second)))
	assert.False(t, d.IsValidAt(now.Add(25*time.Hour)))
}

func TestDiscount_IsActive(t *testing.T) {
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	d, _ := domain.NewDiscount(10, start, end)

	assert.True(t, d.IsActive(now))
	assert.False(t, d.IsActive(now.Add(2*time.Hour)))
}

func TestDiscount_SafePercentageInt64(t *testing.T) {
	d, _ := domain.NewDiscount(50, now, now.Add(time.Hour))
	pct, err := d.SafePercentageInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(50), pct)
}

func TestDiscount_SafePercentageInt64_Overflow(t *testing.T) {
	huge := new(big.Rat).SetFrac(
		new(big.Int).Exp(big.NewInt(2), big.NewInt(64), nil),
		big.NewInt(1),
	)
	d := domain.ReconstructDiscount(huge, now, now.Add(time.Hour))
	_, err := d.SafePercentageInt64()
	assert.Error(t, err)
}

func TestDiscount_Percentage(t *testing.T) {
	d, _ := domain.NewDiscount(25, now, now.Add(time.Hour))
	expected := new(big.Rat).SetInt64(25)
	assert.True(t, d.Percentage().Cmp(expected) == 0)
}

func TestReconstructDiscount(t *testing.T) {
	start := now
	end := now.Add(time.Hour)
	pctRat := new(big.Rat).SetInt64(30)
	d := domain.ReconstructDiscount(pctRat, start, end)
	assert.Equal(t, start, d.StartDate())
	assert.Equal(t, end, d.EndDate())
	pct, err := d.SafePercentageInt64()
	require.NoError(t, err)
	assert.Equal(t, int64(30), pct)
}
