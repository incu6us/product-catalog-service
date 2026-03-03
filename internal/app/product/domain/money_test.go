package domain_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
)

func TestNewMoney_Valid(t *testing.T) {
	m, err := domain.NewMoney(1999, 100)
	require.NoError(t, err)
	assert.Equal(t, "19.99", m.String())
}

func TestNewMoney_ZeroDenominator(t *testing.T) {
	_, err := domain.NewMoney(100, 0)
	assert.ErrorIs(t, err, domain.ErrZeroDenominator)
}

func TestMoney_Add(t *testing.T) {
	a, _ := domain.NewMoney(100, 100)
	b, _ := domain.NewMoney(200, 100)
	result := a.Add(b)
	assert.Equal(t, "3.00", result.String())
}

func TestMoney_Sub(t *testing.T) {
	a, _ := domain.NewMoney(500, 100)
	b, _ := domain.NewMoney(200, 100)
	result := a.Sub(b)
	assert.Equal(t, "3.00", result.String())
}

func TestMoney_Mul(t *testing.T) {
	a, _ := domain.NewMoney(500, 100) // 5.00
	b, _ := domain.NewMoney(2, 1)     // 2
	result := a.Mul(b)
	assert.Equal(t, "10.00", result.String())
}

func TestMoney_IsZero(t *testing.T) {
	m, _ := domain.NewMoney(0, 1)
	assert.True(t, m.IsZero())

	m2, _ := domain.NewMoney(1, 1)
	assert.False(t, m2.IsZero())
}

func TestMoney_IsNegative(t *testing.T) {
	m, _ := domain.NewMoney(-100, 1)
	assert.True(t, m.IsNegative())

	m2, _ := domain.NewMoney(100, 1)
	assert.False(t, m2.IsNegative())
}

func TestMoney_NumeratorDenominator(t *testing.T) {
	m, _ := domain.NewMoney(1999, 100)
	assert.Equal(t, int64(1999), m.Numerator())
	assert.Equal(t, int64(100), m.Denominator())
}

func TestMoney_SafeNumerator(t *testing.T) {
	m, _ := domain.NewMoney(1999, 100)
	n, err := m.SafeNumerator()
	require.NoError(t, err)
	assert.Equal(t, int64(1999), n)
}

func TestMoney_SafeDenominator(t *testing.T) {
	m, _ := domain.NewMoney(1999, 100)
	d, err := m.SafeDenominator()
	require.NoError(t, err)
	assert.Equal(t, int64(100), d)
}

func TestMoney_SafeNumerator_Overflow(t *testing.T) {
	// Create a Money with a huge numerator via MoneyFromRat
	huge := new(big.Rat).SetFrac(
		new(big.Int).Exp(big.NewInt(2), big.NewInt(64), nil),
		big.NewInt(1),
	)
	m := domain.MoneyFromRat(huge)
	_, err := m.SafeNumerator()
	assert.Error(t, err)
}

func TestMoney_Rat(t *testing.T) {
	m, _ := domain.NewMoney(1999, 100)
	r := m.Rat()
	expected := new(big.Rat).SetFrac64(1999, 100)
	assert.True(t, r.Cmp(expected) == 0)
}

func TestMoneyFromRat_Nil(t *testing.T) {
	assert.Panics(t, func() {
		domain.MoneyFromRat(nil)
	})
}

func TestMoney_Numerator_Overflow(t *testing.T) {
	huge := new(big.Rat).SetFrac(
		new(big.Int).Exp(big.NewInt(2), big.NewInt(64), nil),
		big.NewInt(1),
	)
	m := domain.MoneyFromRat(huge)
	assert.Panics(t, func() {
		m.Numerator()
	})
}
