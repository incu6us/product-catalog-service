package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	dto, err := h.app.Queries.GetProduct.Execute(ctx, req.ProductId)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return productDTOToProto(dto), nil
}
