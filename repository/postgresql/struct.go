package postgresql

import (
	"database/sql"
	"time"

	"github.com/Staspol216/gh1/models/order"
)

type orderDTO struct {
	ID             int64             `db:"id"`
	RecipientID    int64             `db:"recipient_id"`
	ExpirationDate time.Time         `db:"expiration_date"`
	DeliveredDate  sql.NullTime      `db:"delivered_date"`
	RefundedDate   sql.NullTime      `db:"refunded_date"`
	ReturnedDate   sql.NullTime      `db:"returned_date"`
	Status         order.OrderStatus `db:"status"`
	Weight         float64           `db:"weight"`
	Worth          float64           `db:"worth"`
}

func transformOrderDtoToModel(o *orderDTO) *order.Order {
	orderModel := &order.Order{
		ID:             o.ID,
		RecipientID:    o.RecipientID,
		ExpirationDate: o.ExpirationDate,
		Status:         o.Status,
		History:        make([]order.OrderRecord, 0),
		Weight:         o.Weight,
		Worth:          o.Worth,
	}
	if o.DeliveredDate.Valid {
		orderModel.DeliveredDate = &o.DeliveredDate.Time
	}
	if o.RefundedDate.Valid {
		orderModel.RefundedDate = &o.RefundedDate.Time
	}
	if o.ReturnedDate.Valid {
		orderModel.ReturnedDate = &o.ReturnedDate.Time
	}
	return orderModel
}

type orderRecordDTO struct {
	ID          int64             `db:"id"`
	OrderID     int64             `db:"order_id"`
	Timestamp   time.Time         `db:"timestamp"`
	Status      order.OrderStatus `db:"status"`
	Description string            `db:"description"`
}

func transformOrderRecordDtoToModel(record *orderRecordDTO) *order.OrderRecord {
	orderRecordModel := &order.OrderRecord{
		Timestamp:   record.Timestamp,
		Status:      record.Status,
		Description: record.Description,
	}

	return orderRecordModel
}
