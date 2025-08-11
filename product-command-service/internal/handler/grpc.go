package handler

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-command-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/service"
	pb "github.com/alimikegami/pos-microservices/proto-defs/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcHandler struct {
	productService service.ProductService
	pb.UnimplementedProductCommandServiceServer
}

func CreateGRPCHandler(productService service.ProductService) *GrpcHandler {
	return &GrpcHandler{
		productService: productService,
	}
}

func (h *GrpcHandler) UpdateProductQuantityBatch(ctx context.Context, req *pb.UpdateProductQuantityRequest) (*emptypb.Empty, error) {
	var orderItem []dto.OrderItem

	for _, item := range req.Products {
		orderItem = append(orderItem, dto.OrderItem{
			ProductID: item.ProductId,
			Quantity:  int(item.Quantity),
		})
	}

	err := h.productService.UpdateProductsQuantity(ctx, dto.OrderRequest{
		OrderItems: orderItem,
	})

	return &emptypb.Empty{}, err
}
