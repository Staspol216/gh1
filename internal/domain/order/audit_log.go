package pvz_domain

import "time"

type OrderOutboxTaskStatus = string

const (
	Processing OrderOutboxTaskStatus = "received"
	Created    OrderOutboxTaskStatus = "created"
	Failed     OrderOutboxTaskStatus = "failed"
)

type OrderOutboxTask struct {
	ID        *int64                `json:"id"`
	Status    OrderOutboxTaskStatus `json:"status"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`

	RequestID     string `json:"request_id,omitempty"`
	Method        string `json:"method,omitempty"`
	Path          string `json:"path,omitempty"`
	RemoteAddress string `json:"remote_address,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`

	OrderStatusDetails *OrderRecord `json:"order_status_details,omitempty"`
}

func (t *OrderOutboxTask) SetOrderStatusDetails(orderRecord *OrderRecord) {
	t.OrderStatusDetails = orderRecord
}
