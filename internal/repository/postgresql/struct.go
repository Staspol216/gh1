package postgresql

import (
	"database/sql"
	"time"

	pvz_model "github.com/Staspol216/gh1/internal/models/order"
)

type orderDTO struct {
	ID             int64                 `db:"id"`
	RecipientID    int64                 `db:"recipient_id"`
	ExpirationDate time.Time             `db:"expiration_date"`
	DeliveredDate  sql.NullTime          `db:"delivered_date"`
	RefundedDate   sql.NullTime          `db:"refunded_date"`
	ReturnedDate   sql.NullTime          `db:"returned_date"`
	Status         pvz_model.OrderStatus `db:"status"`
	Weight         float64               `db:"weight"`
	Worth          float64               `db:"worth"`
}

func transformOrderDtoToModel(o *orderDTO) *pvz_model.Order {
	orderModel := &pvz_model.Order{
		ID:             o.ID,
		RecipientID:    o.RecipientID,
		ExpirationDate: o.ExpirationDate,
		Status:         o.Status,
		History:        make([]pvz_model.OrderRecord, 0),
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
	ID          int64                 `db:"id"`
	OrderID     int64                 `db:"order_id"`
	Timestamp   time.Time             `db:"timestamp"`
	Status      pvz_model.OrderStatus `db:"status"`
	Description string                `db:"description"`
}

func transformOrderRecordDtoToModel(record *orderRecordDTO) *pvz_model.OrderRecord {
	orderRecordModel := &pvz_model.OrderRecord{
		Timestamp:   record.Timestamp,
		Status:      record.Status,
		Description: record.Description,
	}

	return orderRecordModel
}
