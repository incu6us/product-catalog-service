package product

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/incu6us/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/incu6us/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

func productDTOToProto(dto *get_product.ProductDTO) *pb.GetProductReply {
	reply := &pb.GetProductReply{
		ProductId:      dto.ID,
		Name:           dto.Name,
		Description:    dto.Description,
		Category:       dto.Category,
		BasePrice:      dto.BasePrice,
		EffectivePrice: dto.EffectivePrice,
		Status:         dto.Status,
		CreatedAt:      timestamppb.New(dto.CreatedAt),
		UpdatedAt:      timestamppb.New(dto.UpdatedAt),
	}
	if dto.DiscountPercent != nil {
		reply.DiscountPercent = dto.DiscountPercent
	}
	return reply
}

func productListItemToProto(item *list_products.ProductItem) *pb.ProductListItem {
	return &pb.ProductListItem{
		ProductId:      item.ID,
		Name:           item.Name,
		Category:       item.Category,
		BasePrice:      item.BasePrice,
		EffectivePrice: item.EffectivePrice,
		Status:         item.Status,
		CreatedAt:      timestamppb.New(item.CreatedAt),
	}
}
