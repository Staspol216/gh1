package order

import "time"

type Order struct {
	id int64
	expiration time.Time
}

func New(id int64, expiration time.Time) *Order {
	return &Order{
		id,
		expiration,
	}
}

func (o *Order) isExpiredOrder() time.Time {
	return o.expiration
}