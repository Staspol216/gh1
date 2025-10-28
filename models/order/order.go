package order

import "time"

type Order struct {
	ID int64
	Expiration time.Time
}

func New(id int64, expiration time.Time) *Order {
	return &Order{
		id,
		expiration,
	}
}

func (o *Order) IsExpiredOrder() time.Time {
	return o.Expiration
}