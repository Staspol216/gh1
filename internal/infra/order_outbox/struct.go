package order_outbox

import (
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
)

type OrderOutboxTaskStatus = string

const (
	Created    OrderOutboxTaskStatus = "created"
	Processing OrderOutboxTaskStatus = "processing"
	Failed     OrderOutboxTaskStatus = "failed"
)

type OrderOutboxTask struct {
	ID        int64                 `json:"id"`
	Status    OrderOutboxTaskStatus `json:"status"`
	CreatedAt time.Time             `json:"created_at"`

	OrderStatus pvz_domain.OrderStatus `json:"order_status"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
}

func (t *OrderOutboxTask) SetOrderStatusDetails(orderRecord *pvz_domain.OrderRecord) {
	t.OrderStatus = orderRecord.Status
	t.Description = orderRecord.Description
	t.Timestamp = orderRecord.Timestamp
}
