package models

type Order struct {
	id int64
	recipientId int64
	expiration string
}