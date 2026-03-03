package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/update_product"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := h.app.Commands.UpdateProduct.Execute(ctx, update_product.Request{
		ProductID:   req.ProductId,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.UpdateProductReply{}, nil
}
