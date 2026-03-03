package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Discount represents a percentage-based discount with a validity period.
type Discount struct {
	percentage *big.Rat
	startDate  time.Time
	endDate    time.Time
}

// NewDiscount creates a Discount with the given percentage (0-100) and date range.
func NewDiscount(percentage int64, start, end time.Time) (*Discount, error) {
	if percentage <= 0 || percentage > 100 {
		return nil, ErrInvalidDiscountPercentage
	}
	if !end.After(start) {
		return nil, ErrInvalidDiscountPeriod
	}
	return &Discount{
		percentage: new(big.Rat).SetInt64(percentage),
		startDate:  start,
		endDate:    end,
	}, nil
}

// ReconstructDiscount creates a Discount from stored data without validation.
func ReconstructDiscount(percentage *big.Rat, start, end time.Time) *Discount {
	return &Discount{
		percentage: percentage,
		startDate:  start,
		endDate:    end,
	}
}

// IsValidAt returns true if t is within the discount's validity period.
func (d *Discount) IsValidAt(t time.Time) bool {
	return !t.Before(d.startDate) && !t.After(d.endDate)
}

// IsActive is an alias for IsValidAt.
func (d *Discount) IsActive(t time.Time) bool {
	return d.IsValidAt(t)
}

// Percentage returns the discount percentage as *big.Rat.
func (d *Discount) Percentage() *big.Rat {
	return new(big.Rat).Set(d.percentage)
}

// StartDate returns the start date of the discount.
func (d *Discount) StartDate() time.Time {
	return d.startDate
}

// EndDate returns the end date of the discount.
func (d *Discount) EndDate() time.Time {
	return d.endDate
}

// SafePercentageInt64 returns the discount percentage as an int64,
// returning an error if the numerator overflows int64.
func (d *Discount) SafePercentageInt64() (int64, error) {
	num := d.percentage.Num()
	if !num.IsInt64() {
		return 0, fmt.Errorf("discount percentage overflows int64: %s", num.String())
	}
	return num.Int64(), nil
}
