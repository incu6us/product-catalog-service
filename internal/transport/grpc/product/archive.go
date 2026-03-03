package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/app/product/usecases/archive_product"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) ArchiveProduct(ctx context.Context, req *pb.ArchiveProductRequest) (*pb.ArchiveProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	err := h.app.Commands.ArchiveProduct.Execute(ctx, archive_product.Request{
		ProductID: req.ProductId,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.ArchiveProductReply{}, nil
}
