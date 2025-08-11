package handler

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-query-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/service"
	pkgdto "github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
	"github.com/alimikegami/pos-microservices/proto-defs/pb"
)

type GrpcHandler struct {
	productService service.ProductService
	pb.UnimplementedProductQueryServiceServer
}

func CreateGRPCHandler(productService service.ProductService) *GrpcHandler {
	return &GrpcHandler{
		productService: productService,
	}
}

func (h *GrpcHandler) GetProductPrice(_ context.Context, req *pb.GetProductPriceRequest) (*pb.ProductPriceResponse, error) {
	response, err := h.productService.GetProducts(context.Background(), pkgdto.Filter{
		ProductIds: req.GetProductIds(),
	})

	if err != nil {
		return nil, err
	}

	var products []*pb.Product
	for _, product := range response.Records.([]dto.ProductResponse) {
		products = append(products, &pb.Product{
			ProductId: product.ID,
			Name:      product.Name,
			Price:     float32(product.Price),
			Quantity:  int64(product.Quantity),
		})
	}

	return &pb.ProductPriceResponse{
		Products: products,
	}, nil
}
