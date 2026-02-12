package pvz_domain

import "time"

const (
	OrderStatusReceived  OrderStatus = "received"
	OrderStatusReturned  OrderStatus = "returned"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusRefunded  OrderStatus = "refunded"
	OrderStatusExpired   OrderStatus = "storage_ended"
	OrderStatusNone      OrderStatus = "none"
)

var OrderStatusDescription = map[OrderStatus]string{
	OrderStatusReceived:  "Заказ получен от курьера",
	OrderStatusRefunded:  "Заказ возвращен от клиента",
	OrderStatusDelivered: "Заказ выдан клиенту",
	OrderStatusExpired:   "Срок хранения заказа истек",
	OrderStatusNone:      "",
}

type OrderRecord struct {
	Timestamp   time.Time   `json:"timestamp"`
	Status      OrderStatus `json:"status"`
	Description string      `json:"description"`
}

func NewOrderRecordReceived() *OrderRecord {
	Status := OrderStatusReceived
	return &OrderRecord{
		Timestamp:   time.Now(),
		Status:      Status,
		Description: OrderStatusDescription[Status],
	}
}

func NewOrderRecordRefunded() *OrderRecord {
	Status := OrderStatusRefunded
	return &OrderRecord{
		Timestamp:   time.Now(),
		Status:      Status,
		Description: OrderStatusDescription[Status],
	}
}

func NewOrderRecordDelivered() *OrderRecord {
	Status := OrderStatusDelivered
	return &OrderRecord{
		Timestamp:   time.Now(),
		Status:      Status,
		Description: OrderStatusDescription[Status],
	}
}

func NewOrderRecordExpired() *OrderRecord {
	Status := OrderStatusExpired
	return &OrderRecord{
		Timestamp:   time.Now(),
		Status:      Status,
		Description: OrderStatusDescription[Status],
	}
}
