package pvz_domain

import "time"

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

	Order_status OrderStatus `json:"order_status"`
	Description  string      `json:"description"`
	Timestamp    time.Time   `json:"timestamp"`
}

func (t *OrderOutboxTask) SetOrderStatusDetails(orderRecord *OrderRecord) {
	t.Order_status = orderRecord.Status
	t.Description = orderRecord.Description
	t.Timestamp = orderRecord.Timestamp
}
