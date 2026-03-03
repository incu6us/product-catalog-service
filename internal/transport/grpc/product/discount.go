package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/remove_discount"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) ApplyDiscount(ctx context.Context, req *pb.ApplyDiscountRequest) (*pb.ApplyDiscountReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.Percentage <= 0 || req.Percentage > 100 {
		return nil, status.Error(codes.InvalidArgument, "percentage must be between 1 and 100")
	}
	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	err := h.app.Commands.ApplyDiscount.Execute(ctx, apply_discount.Request{
		ProductID:  req.ProductId,
		Percentage: req.Percentage,
		StartDate:  req.StartDate.AsTime(),
		EndDate:    req.EndDate.AsTime(),
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.ApplyDiscountReply{}, nil
}

func (h *Handler) RemoveDiscount(ctx context.Context, req *pb.RemoveDiscountRequest) (*pb.RemoveDiscountReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	err := h.app.Commands.RemoveDiscount.Execute(ctx, remove_discount.Request{
		ProductID: req.ProductId,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.RemoveDiscountReply{}, nil
}
