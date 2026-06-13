package orders_http_handler

import (
	"fmt"

	"github.com/Staspol216/gh1/internal/domain/order"
	orders_dto "github.com/Staspol216/gh1/pkg/api/openapi/orders"
)

func toOrderDTO(order *pvz_domain.Order) orders_dto.Order {

	history := make([]orders_dto.OrderRecord, 0, len(order.History))

	for _, v := range order.History {
		orderRecord := toOrderRecordDTO(v)

		history = append(history, orderRecord)
	}

	status, err := toOrderStatusDTO(order.Status)
	if err != nil {
		panic(err)
	}

	return orders_dto.Order{
		DeliveredDate:  order.DeliveredDate,
		ExpirationDate: order.ExpirationDate,
		History:        history,
		Id:             order.ID,
		RecipientId:    order.RecipientID,
		RefundedDate:   order.RefundedDate,
		ReturnedDate:   order.ReturnedDate,
		Status:         status,
		Weight:         order.Weight,
		Worth:          order.Worth,
	}
}

func toOrderRecordDTO(orderRecord pvz_domain.OrderRecord) orders_dto.OrderRecord {

	status, err := toOrderStatusDTO(orderRecord.Status)

	if err != nil {
		panic(err)
	}

	return orders_dto.OrderRecord{
		Description: orderRecord.Description,
		Status:      status,
		Timestamp:   orderRecord.Timestamp,
	}
}

func toOrderStatusDTO(orderStatus pvz_domain.OrderStatus) (orders_dto.OrderStatus, error) {
	switch orderStatus {
	case pvz_domain.OrderStatusDelivered:
		return orders_dto.Delivered, nil
	case pvz_domain.OrderStatusRefunded:
		return orders_dto.Refunded, nil
	case pvz_domain.OrderStatusReturned:
		return orders_dto.Returned, nil
	case pvz_domain.OrderStatusReceived:
		return orders_dto.Received, nil
	case pvz_domain.OrderStatusNone:
		return orders_dto.None, nil
	default:
		return orders_dto.None, fmt.Errorf("unknown order status: %v", orderStatus)
	}
}
