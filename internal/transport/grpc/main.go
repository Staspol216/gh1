package pvz_grpc

import (
	"context"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/service/order"
	"github.com/Staspol216/gh1/pkg/api/orders.proto"
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
	service *pvz_order_service.PvzService
	orders_proto.UnimplementedOrdersServiceServer
}

func New(p *pvz_order_service.PvzService) *GrpcHandler {
	return &GrpcHandler{
		service: p,
	}
}

var _ orders_proto.OrdersServiceServer = (*GrpcHandler)(nil)

func (s *GrpcHandler) GetOrders(ctx context.Context, req *orders_proto.GetOrdersRequest) (resp *orders_proto.GetOrdersResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("GetOrders", err, time.Since(startTime))
	}()

	pagination := &pvz_domain.Pagination{
		Offset: req.GetOffset(),
		Limit:  req.GetLimit(),
	}

	orders, err := s.service.GetOrders(ctx, pagination)

	if err != nil {
		app_logger.MyLogger.Error("gRPC GetOrders failed",
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("limit", req.GetLimit()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.GetOrdersResponse{
		Orders: NewOrdersListResponse(orders),
	}, nil
}

func (s *GrpcHandler) GetOrder(ctx context.Context, req *orders_proto.GetOrderRequest) (resp *orders_proto.Order, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("GetOrder", err, time.Since(startTime))
	}()

	order, err := s.service.GetOrderByID(ctx, req.GetOrderId(), req.GetRecipientId())
	if err != nil {
		app_logger.MyLogger.Error("gRPC GetOrder failed",
			zap.Int64("order_id", req.GetOrderId()),
			zap.Int64("recipient_id", req.GetRecipientId()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return mapDomainOrderToProtoOrder(order), nil
}

func (s *GrpcHandler) GetRefundedOrders(ctx context.Context, req *orders_proto.GetOrdersRequest) (resp *orders_proto.GetOrdersResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("GetRefundedOrders", err, time.Since(startTime))
	}()

	pagination := &pvz_domain.Pagination{
		Offset: req.GetOffset(),
		Limit:  req.GetLimit(),
	}

	orders, err := s.service.GetAllRefunds(ctx, pagination)
	if err != nil {
		app_logger.MyLogger.Error("gRPC GetRefundedOrders failed",
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("limit", req.GetLimit()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.GetOrdersResponse{
		Orders: NewOrdersListResponse(orders),
	}, nil
}

func (s *GrpcHandler) GetOrdersHistory(ctx context.Context, req *orders_proto.GetOrdersRequest) (resp *orders_proto.GetOrdersResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("GetOrdersHistory", err, time.Since(startTime))
	}()

	pagination := &pvz_domain.Pagination{
		Offset: req.GetOffset(),
		Limit:  req.GetLimit(),
	}

	orders, err := s.service.GetHistory(ctx, pagination)
	if err != nil {
		app_logger.MyLogger.Error("gRPC GetOrdersHistory failed",
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("limit", req.GetLimit()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.GetOrdersResponse{
		Orders: NewOrdersListResponse(orders),
	}, nil
}

func (s *GrpcHandler) CreateOrder(ctx context.Context, req *orders_proto.OrderCreateRequest) (resp *orders_proto.OrderIDResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("CreateOrder", err, time.Since(startTime))
	}()

	order := mapToDomainOrderParams(req.GetOrder())

	packagingType := req.GetPackagingType().String()
	orderId, err := s.service.AcceptFromCourier(ctx, order, packagingType, req.GetMembranaIncluded())

	if err != nil {
		app_logger.MyLogger.Error("gRPC CreateOrder failed",
			zap.String("packaging_type", packagingType),
			zap.Bool("membrana_included", req.GetMembranaIncluded()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.OrderIDResponse{
		OrderId: *orderId,
	}, nil
}

func (s *GrpcHandler) UpdateOrders(ctx context.Context, req *orders_proto.OrderUpdateRequest) (resp *orders_proto.UpdateOrdersResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("UpdateOrders", err, time.Since(startTime))
	}()

	action := req.GetAction().String()
	err = s.service.ServeRecipient(ctx, req.GetOrderIds(), req.GetRecipientId(), action)

	if err != nil {
		app_logger.MyLogger.Error("gRPC UpdateOrders failed",
			zap.Int64s("order_ids", req.GetOrderIds()),
			zap.Int64("recipient_id", req.GetRecipientId()),
			zap.String("action", action),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.UpdateOrdersResponse{}, nil
}

func (s *GrpcHandler) DeleteOrder(ctx context.Context, req *orders_proto.DeleteOrderRequest) (resp *orders_proto.DeleteOrderResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("DeleteOrder", err, time.Since(startTime))
	}()

	err = s.service.ReturnToCourier(ctx, req.GetOrderId())

	if err != nil {
		app_logger.MyLogger.Error("gRPC DeleteOrder failed",
			zap.Int64("order_id", req.GetOrderId()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.DeleteOrderResponse{}, nil
}

func (s *GrpcHandler) GetPing(ctx context.Context, req *orders_proto.GetPingRequest) (resp *orders_proto.GetPingResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("GetPing", err, time.Since(startTime))
	}()

	return &orders_proto.GetPingResponse{
		Message: "pong",
	}, nil
}

func NewOrdersListResponse(orders []*pvz_domain.Order) []*orders_proto.Order {
	list := make([]*orders_proto.Order, 0, len(orders))
	for _, order := range orders {
		list = append(list, mapDomainOrderToProtoOrder(order))
	}
	return list
}

func timePtrToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
