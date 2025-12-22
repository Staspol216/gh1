package pvz_model

import (
	"time"
)

type OrderStatus string

const (
	OrderStatusReceived  OrderStatus = "received"      // принят от курьера
	OrderStatusReturned  OrderStatus = "returned"      // возвращен курьеру
	OrderStatusDelivered OrderStatus = "delivered"     // выдан клиенту
	OrderStatusRefunded  OrderStatus = "refunded"      // возвращен клиентом
	OrderStatusExpired   OrderStatus = "storage_ended" // срок хранения истек
)

type Order struct {
	ID             int64         `json:"id"`
	RecipientID    int64         `json:"recipient_id"`
	ExpirationDate time.Time     `json:"expiration_date"`
	DeliveredDate  *time.Time    `json:"delivered_date"`
	RefundedDate   *time.Time    `json:"refunded_date"`
	ReturnedDate   *time.Time    `json:"returned_date"`
	Status         OrderStatus   `json:"status"`
	History        []OrderRecord `json:"history"`
	Weight         float64       `json:"weight"`
	Worth          float64       `json:"worth"`
}

type OrderRecord struct {
	Timestamp   time.Time   `json:"timestamp"`
	Status      OrderStatus `json:"status"`
	Description string      `json:"description"`
}

type OrderParams struct {
	OrderId        int64     `json:"order_id"`
	RecipientId    int64     `json:"recipient_id"`
	ExpirationDate time.Time `json:"expiration_date"`
	Weight         float64   `json:"weight"`
	Worth          float64   `json:"worth"`
}

func New(data *OrderParams) *Order {
	return &Order{
		ID:             data.OrderId,
		ExpirationDate: data.ExpirationDate,
		RecipientID:    data.RecipientId,
		Status:         OrderStatusReceived,
		History:        []OrderRecord{},
		Weight:         data.Weight,
		Worth:          data.Worth,
	}
}

func (o *Order) IsRefunded() bool {
	return o.Status == OrderStatusRefunded
}

func (o *Order) IsDelivered() bool {
	return o.Status == OrderStatusDelivered
}

func (o *Order) IsRecieved() bool {
	return o.Status == OrderStatusReceived
}

func (o *Order) CanBeRefunded() bool {
	const DAYS_FOR_REFUNDING = 2
	refundExpirationDate := o.DeliveredDate.AddDate(0, 0, DAYS_FOR_REFUNDING)
	canBeRefunded := o.DeliveredDate.Compare(refundExpirationDate) == -1
	return canBeRefunded
}

func (o *Order) Refund() {
	now := time.Now()
	o.RefundedDate = &now
}

func (o *Order) IsExpired() bool {
	res := o.ExpirationDate.Compare(time.Now())
	return res == -1
}

func (o *Order) SetStatus(status OrderStatus) {
	o.Status = status
}

func (o *Order) SetDeliveredDate(date *time.Time) {
	o.DeliveredDate = date
}
