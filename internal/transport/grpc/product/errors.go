package product

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/pkg/committer"
)

func mapDomainErrorToGRPC(err error) error {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrProductNotActive),
		errors.Is(err, domain.ErrProductAlreadyActive),
		errors.Is(err, domain.ErrProductAlreadyArchived),
		errors.Is(err, domain.ErrProductHasActiveDiscount),
		errors.Is(err, domain.ErrNoDiscountToRemove):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, domain.ErrInvalidDiscountPeriod),
		errors.Is(err, domain.ErrInvalidDiscountPercentage),
		errors.Is(err, domain.ErrInvalidProductName),
		errors.Is(err, domain.ErrInvalidCategory),
		errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrZeroDenominator):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, committer.ErrConcurrentModification):
		return status.Error(codes.Aborted, err.Error())

	default:
		return status.Error(codes.Internal, "internal error")
	}
}
