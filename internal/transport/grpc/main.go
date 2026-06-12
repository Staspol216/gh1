package pvz_grpc

import (
	"context"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
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

func (s *GrpcHandler) CreateOrder(ctx context.Context, req *orders_proto.CreateOrderRequest) (resp *orders_proto.CreateOrderResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("CreateOrder", err, time.Since(startTime))
	}()

	order := mapToDomainOrderParams(req.GetOrder())

	orderId, err := s.service.AcceptFromCourier(ctx, order, req.GetPackagingType(), req.GetMembranaIncluded())

	if err != nil {
		app_logger.MyLogger.Error("gRPC CreateOrder failed",
			zap.String("packaging_type", req.GetPackagingType()),
			zap.Bool("membrana_included", req.GetMembranaIncluded()),
			zap.Error(err),
		)
		err = status.Errorf(codes.Internal, "Internal service error: %s", err)
		return nil, err
	}

	return &orders_proto.CreateOrderResponse{
		OrderId: *orderId,
	}, nil
}

func (s *GrpcHandler) UpdateOrders(ctx context.Context, req *orders_proto.UpdateOrdersRequest) (resp *orders_proto.UpdateOrdersResponse, err error) {
	startTime := time.Now()
	defer func() {
		monitoring.ObserveGRPCRequest("UpdateOrders", err, time.Since(startTime))
	}()

	err = s.service.ServeRecipient(ctx, req.GetOrderIds(), req.GetRecipientId(), req.GetAction())

	if err != nil {
		app_logger.MyLogger.Error("gRPC UpdateOrders failed",
			zap.Int64s("order_ids", req.GetOrderIds()),
			zap.Int64("recipient_id", req.GetRecipientId()),
			zap.String("action", req.GetAction()),
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

func (s *GrpcHandler) createOutboxTask() *order_outbox.OrderOutboxTask {

	createdAt := time.Now()

	log := &order_outbox.OrderOutboxTask{
		Status:    order_outbox.Created,
		CreatedAt: createdAt,
	}

	return log
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
