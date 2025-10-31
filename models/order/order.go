package order

import "time"

type Order struct {
	ID          int64     `json:"id"`
	Expiration  time.Time `json:"expiration"`
	RecipientId int64     `json:"recipient_id"`
}

func New(id int64, recipientId int64, expiration time.Time) *Order {
	return &Order{
		ID:          id,
		Expiration:  expiration,
		RecipientId: recipientId,
	}
}

func (o *Order) IsExpired() bool {
	res := o.Expiration.Compare(time.Now())
	return res == 1
}
