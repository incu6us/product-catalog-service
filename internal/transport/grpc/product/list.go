package product

import (
	"context"

	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func (h *Handler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsReply, error) {
	result, err := h.app.Queries.ListProducts.Execute(ctx, list_products.ListParams{
		Category:  req.Category,
		Status:    req.Status,
		PageSize:  int(req.PageSize),
		PageToken: req.PageToken,
	})
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	reply := &pb.ListProductsReply{
		NextPageToken: result.NextPageToken,
		Products:      make([]*pb.ProductListItem, 0, len(result.Products)),
	}
	for _, item := range result.Products {
		reply.Products = append(reply.Products, productListItemToProto(item))
	}

	return reply, nil
}
