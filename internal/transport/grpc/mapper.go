package pvz_grpc

import (
	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/pkg/api/orders.proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapStatusToProto(s pvz_domain.OrderStatus) orders_proto.OrderStatus {
	switch s {
	case pvz_domain.OrderStatusReceived:
		return orders_proto.OrderStatus_received
	case pvz_domain.OrderStatusReturned:
		return orders_proto.OrderStatus_returned
	case pvz_domain.OrderStatusRefunded:
		return orders_proto.OrderStatus_refunded
	case pvz_domain.OrderStatusDelivered:
		return orders_proto.OrderStatus_delivered
	case pvz_domain.OrderStatusExpired:
		return orders_proto.OrderStatus_storage_ended
	case pvz_domain.OrderStatusNone:
		return orders_proto.OrderStatus_none
	default:
		return orders_proto.OrderStatus_none
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
		ReturnedDate:   timePtrToProto(o.ReturnedDate),
		Status:         mapStatusToProto(o.Status),
		History:        mapHistoryToProto(o.History),
		Weight:         o.Weight,
		Worth:          o.Worth,
	}
}

func mapToDomainOrderParams(p *orders_proto.OrderCreateRequest_OrderParams) *pvz_domain.OrderParams {
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
