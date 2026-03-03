package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/deactivate_product"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) ActivateProduct(ctx context.Context, req *pb.ActivateProductRequest) (*pb.ActivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	err := h.app.Commands.ActivateProduct.Execute(ctx, activate_product.Request{
		ProductID: req.ProductId,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.ActivateProductReply{}, nil
}

func (h *Handler) DeactivateProduct(ctx context.Context, req *pb.DeactivateProductRequest) (*pb.DeactivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	err := h.app.Commands.DeactivateProduct.Execute(ctx, deactivate_product.Request{
		ProductID: req.ProductId,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.DeactivateProductReply{}, nil
}
