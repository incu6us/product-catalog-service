package product

import (
	"github.com/incu6us/product-catalog-service/internal/services"
	"github.com/incu6us/product-catalog-service/internal/transport/grpc/product/pb"
)

// Handler implements the ProductService gRPC server.
type Handler struct {
	pb.UnimplementedProductServiceServer
	app *services.App
}

// NewHandler creates a new gRPC product handler.
func NewHandler(app *services.App) *Handler {
	return &Handler{app: app}
}
