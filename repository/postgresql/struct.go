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

func transformDtoToModel(o *orderDTO) *order.Order {
	oRepo := &order.Order{
		ID:             o.ID,
		RecipientID:    o.RecipientID,
		ExpirationDate: o.ExpirationDate,
		Status:         o.Status,
		Weight:         o.Weight,
		Worth:          o.Worth,
	}
	if o.DeliveredDate.Valid {
		oRepo.DeliveredDate = &o.DeliveredDate.Time
	}
	if o.RefundedDate.Valid {
		oRepo.RefundedDate = &o.RefundedDate.Time
	}
	if o.ReturnedDate.Valid {
		oRepo.ReturnedDate = &o.ReturnedDate.Time
	}
	return oRepo
}
