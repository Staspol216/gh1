package order

import "time"

type Order struct {
	ID int64				`json:"id"`
	Expiration time.Time	`json:"expiration"`
}

func New(id int64, expiration time.Time) *Order {
	return &Order{
		id,
		expiration,
	}
}

func (o *Order) IsExpiredOrder() bool {
	res := o.Expiration.Compare(time.Now())
	return res == 1
}