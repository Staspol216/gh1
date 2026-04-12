package pvz_grpc

import (
	"context"
	"time"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	pvz_order_service "github.com/Staspol216/gh1/internal/service/order"
	orders_proto "github.com/Staspol216/gh1/pkg/api/orders.proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcHandler struct {
	context context.Context
	service *pvz_order_service.Pvz
	orders_proto.UnimplementedOrdersServiceServer
}

func New(context context.Context, p *pvz_order_service.Pvz) *GrpcHandler {
	return &GrpcHandler{
		context: context,
		service: p,
	}
}

func (s *GrpcHandler) GetOrders(ctx context.Context, req *orders_proto.GetOrdersRequest) (*orders_proto.GetOrdersResponse, error) {

	pagination := &pvz_domain.Pagination{
		Offset: req.GetOffset(),
		Limit:  req.GetLimit(),
	}

	orders, err := s.service.GetOrders(ctx, pagination)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal service error: %s", err)
	}

	return &orders_proto.GetOrdersResponse{
		Orders: NewOrdersListResponse(orders),
	}, nil
}

func (s *GrpcHandler) CreateOrder(ctx context.Context, req *orders_proto.CreateOrderRequest) (*orders_proto.CreateOrderResponse, error) {

	task := s.createOutboxTask()

	order := mapToDomainOrderParams(req.GetOrder())

	orderId, err := s.service.AcceptFromCourier(s.context, task, order, req.GetPackagingType(), req.GetMembranaIncluded())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal service error: %s", err)
	}

	return &orders_proto.CreateOrderResponse{
		OrderId: *orderId,
	}, nil
}

func (s *GrpcHandler) UpdateOrders(ctx context.Context, req *orders_proto.UpdateOrdersRequest) (*orders_proto.UpdateOrdersResponse, error) {

	task := s.createOutboxTask()

	err := s.service.ServeRecipient(s.context, task, req.GetOrderIds(), req.GetRecipientId(), req.GetAction())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal service error: %s", err)
	}

	return &orders_proto.UpdateOrdersResponse{}, nil
}

func (s *GrpcHandler) DeleteOrder(ctx context.Context, req *orders_proto.DeleteOrderRequest) (*orders_proto.DeleteOrderResponse, error) {

	err := s.service.ReturnToCourier(s.context, req.GetOrderId())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal service error: %s", err)
	}

	return &orders_proto.DeleteOrderResponse{}, nil
}

func (s *GrpcHandler) createOutboxTask() *pvz_domain.OrderOutboxTask {
	requestID := uuid.New().String()

	createdAt := time.Now()

	log := &pvz_domain.OrderOutboxTask{
		Status:    pvz_domain.Created,
		CreatedAt: createdAt,
		RequestID: requestID,
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

func mapStatusToProto(s pvz_domain.OrderStatus) orders_proto.OrderStatus {
	switch s {
	case pvz_domain.OrderStatusReceived:
		return orders_proto.OrderStatus_RECEVIED
	case pvz_domain.OrderStatusReturned:
		return orders_proto.OrderStatus_RETURNED
	case pvz_domain.OrderStatusRefunded:
		return orders_proto.OrderStatus_REFUNDED
	case pvz_domain.OrderStatusDelivered:
		return orders_proto.OrderStatus_DELIVERED
	case pvz_domain.OrderStatusExpired:
		return orders_proto.OrderStatus_STRAGE_ENDED
	case pvz_domain.OrderStatusNone:
		return orders_proto.OrderStatus_NONE
	default:
		return orders_proto.OrderStatus_NONE
	}
}

func mapHistoryToProto(records []pvz_domain.OrderRecord) []*orders_proto.OrderRecord {
	result := make([]*orders_proto.OrderRecord, 0, len(records))

	for _, r := range records {
		result = append(result, &orders_proto.OrderRecord{
			Timestamp:   timestamppb.New(r.Timestamp),
			Status:      mapStatusToProto(r.Status),
			Description: r.Description,
		})
	}

	return result
}

func mapDomainOrderToProtoOrder(o *pvz_domain.Order) *orders_proto.Order {
	if o == nil {
		return nil
	}

	return &orders_proto.Order{
		Id:             o.ID,
		RecipientId:    o.RecipientID,
		ExpirationDate: timestamppb.New(o.ExpirationDate),
		DeliveredDate:  timePtrToProto(o.DeliveredDate),
		RefundedDate:   timePtrToProto(o.RefundedDate),
		Status:         mapStatusToProto(o.Status),
		History:        mapHistoryToProto(o.History),
		Weight:         o.Weight,
		Worth:          o.Worth,
	}
}

func mapToDomainOrderParams(p *orders_proto.CreateOrderRequest_OrderParams) *pvz_domain.OrderParams {
	if p == nil {
		return nil
	}

	return &pvz_domain.OrderParams{
		RecipientId:    p.GetRecipientId(),
		ExpirationDate: p.GetExpirationDate().AsTime(),
		Weight:         p.GetWeight(),
		Worth:          p.GetWorth(),
	}
}
