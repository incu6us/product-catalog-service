package domain

import (
	"errors"
	"fmt"
	"math/big"
)

// ErrZeroDenominator is returned when a Money value is created with a zero denominator.
var ErrZeroDenominator = errors.New("money denominator must not be zero")

// Money wraps *big.Rat for precise decimal arithmetic.
type Money struct {
	value *big.Rat
}

// NewMoney creates a Money from a numerator and denominator.
func NewMoney(numerator, denominator int64) (*Money, error) {
	if denominator == 0 {
		return nil, ErrZeroDenominator
	}
	return &Money{value: new(big.Rat).SetFrac64(numerator, denominator)}, nil
}

// MoneyFromRat creates a Money from an existing *big.Rat.
// Panics if r is nil (programming error).
func MoneyFromRat(r *big.Rat) *Money {
	if r == nil {
		panic("MoneyFromRat: nil *big.Rat")
	}
	return &Money{value: new(big.Rat).Set(r)}
}

// Add returns a new Money equal to m + other.
func (m *Money) Add(other *Money) *Money {
	result := new(big.Rat).Add(m.value, other.value)
	return &Money{value: result}
}

// Sub returns a new Money equal to m - other.
func (m *Money) Sub(other *Money) *Money {
	result := new(big.Rat).Sub(m.value, other.value)
	return &Money{value: result}
}

// Mul returns a new Money equal to m * other.
func (m *Money) Mul(other *Money) *Money {
	result := new(big.Rat).Mul(m.value, other.value)
	return &Money{value: result}
}

// Numerator returns the numerator of the underlying fraction.
// Panics if the value overflows int64; use SafeNumerator for checked access.
func (m *Money) Numerator() int64 {
	n, err := m.SafeNumerator()
	if err != nil {
		panic(err)
	}
	return n
}

// Denominator returns the denominator of the underlying fraction.
// Panics if the value overflows int64; use SafeDenominator for checked access.
func (m *Money) Denominator() int64 {
	d, err := m.SafeDenominator()
	if err != nil {
		panic(err)
	}
	return d
}

// Rat returns the underlying *big.Rat (a copy).
func (m *Money) Rat() *big.Rat {
	return new(big.Rat).Set(m.value)
}

// String returns a human-readable representation of the money value.
// Note: converts to float64 for formatting, which may lose precision for very large values.
func (m *Money) String() string {
	f, _ := m.value.Float64()
	return fmt.Sprintf("%.2f", f)
}

// IsZero returns true if the money value is zero.
func (m *Money) IsZero() bool {
	return m.value.Sign() == 0
}

// IsNegative returns true if the money value is negative.
func (m *Money) IsNegative() bool {
	return m.value.Sign() < 0
}

// SafeNumerator returns the numerator, checking that it fits in int64.
func (m *Money) SafeNumerator() (int64, error) {
	num := m.value.Num()
	if !num.IsInt64() {
		return 0, fmt.Errorf("numerator overflows int64: %s", num.String())
	}
	return num.Int64(), nil
}

// SafeDenominator returns the denominator, checking that it fits in int64.
func (m *Money) SafeDenominator() (int64, error) {
	denom := m.value.Denom()
	if !denom.IsInt64() {
		return 0, fmt.Errorf("denominator overflows int64: %s", denom.String())
	}
	return denom.Int64(), nil
}
