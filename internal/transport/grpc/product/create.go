package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductReply, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Category == "" {
		return nil, status.Error(codes.InvalidArgument, "category is required")
	}
	if req.BasePriceNumerator <= 0 {
		return nil, status.Error(codes.InvalidArgument, "base_price_numerator must be positive")
	}
	if req.BasePriceDenominator <= 0 {
		return nil, status.Error(codes.InvalidArgument, "base_price_denominator must be positive")
	}

	productID, err := h.app.Commands.CreateProduct.Execute(ctx, create_product.Request{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		BasePriceNumerator:   req.BasePriceNumerator,
		BasePriceDenominator: req.BasePriceDenominator,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.CreateProductReply{ProductId: productID}, nil
}
