package domain

import "errors"

var (
	ErrProductNotActive          = errors.New("product is not active")
	ErrProductAlreadyActive      = errors.New("product is already active")
	ErrProductNotFound           = errors.New("product not found")
	ErrInvalidDiscountPeriod     = errors.New("invalid discount period")
	ErrInvalidDiscountPercentage = errors.New("invalid discount percentage")
	ErrProductAlreadyArchived    = errors.New("product is already archived")
	ErrProductHasActiveDiscount  = errors.New("product already has an active discount")
	ErrInvalidProductName        = errors.New("invalid product name")
	ErrInvalidCategory           = errors.New("invalid category")
	ErrInvalidPrice              = errors.New("invalid price")
	ErrNoDiscountToRemove        = errors.New("no discount to remove")
)
