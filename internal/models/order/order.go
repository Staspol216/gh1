package pvz_model

import (
	"time"
)

type OrderStatus string

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

type OrderParams struct {
	RecipientId    int64     `json:"recipient_id"`
	ExpirationDate time.Time `json:"expiration_date"`
	Weight         float64   `json:"weight"`
	Worth          float64   `json:"worth"`
}

func New(data *OrderParams) *Order {
	return &Order{
		ExpirationDate: data.ExpirationDate,
		RecipientID:    data.RecipientId,
		Status:         OrderStatusNone,
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

func (o *Order) IsExpired() bool {
	res := o.ExpirationDate.Compare(time.Now())
	return res == -1
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
	o.SetStatus(OrderStatusRefunded)
}

func (o *Order) Deliver() {
	now := time.Now()
	o.DeliveredDate = &now
	o.SetStatus(OrderStatusDelivered)
}

func (o *Order) Expire() {
	o.SetStatus(OrderStatusExpired)
}

func (o *Order) SetStatus(status OrderStatus) {
	o.Status = status
}
