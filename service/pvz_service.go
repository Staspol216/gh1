package Serivces

import (
	"fmt"
	"slices"
	"time"

	"github.com/Staspol216/gh1/models/order"
	"github.com/Staspol216/gh1/storage"
)

type Action int

const (
	Deliver Action = iota
	Refund
)

var ServeRecipientActionName = map[Action]string{
	Deliver: "deliver",
	Refund:  "refund",
}

func (c Action) String() string {
	return ServeRecipientActionName[c]
}

type Pvz struct {
	storage storage.Storager
}

func New(storage storage.Storager) *Pvz {
	return &Pvz{
		storage,
	}
}

func (s *Pvz) AcceptFromCourier(orderId int64, recipientId int64, expiration time.Time) {
	newOrder := order.New(orderId, recipientId, expiration)
	newOrder.SetStatus(order.OrderStatusReceived)
	newOrder.AddHistoryRecord("Заказ получен от курьера")
	s.storage.SaveOrder(newOrder)
}

func (s *Pvz) ReturnToCourier(orderId int64) {
	defer func() {
		err := s.storage.SaveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	order, exists := s.storage.FindOrderById(orderId)

	if exists && order.IsExpired() {
		s.storage.DeleteOrderById(orderId)
	}
}

func (s *Pvz) ServeRecipient(ordersIds []int64, recipientId int64, action string) {
	switch action {
	case Deliver.String():
		s.DeliverOrdersById(ordersIds, recipientId)
	case Refund.String():
		s.RefundOrdersById(ordersIds, recipientId)
	default:
		fmt.Println("Unknown action for ServeRecipient command")
	}
}

func (p *Pvz) RefundOrdersById(orderIds []int64, recipientId int64) {
	defer func() {
		err := p.storage.SaveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	orders := p.storage.FindRecipientOrdersByIds(orderIds, recipientId)

	for _, order := range orders {
		p.RefundOrder(order)
	}
}

func (p *Pvz) RefundOrder(targetOrder *order.Order) {
	if targetOrder.CanBeRefunded() {
		targetOrder.RefundByRecipient()
	} else {
		fmt.Printf("Order %d can not be refunded to recipient because refund time has expired", targetOrder.ID)
	}
}

func (s *Pvz) DeliverOrdersById(orderIds []int64, recipientId int64) {
	defer func() {
		err := s.storage.SaveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	orders := s.storage.FindRecipientOrdersByIds(orderIds, recipientId)

	for _, order := range orders {
		s.DeliverOrder(order)
	}
}

func (p *Pvz) DeliverOrder(targetOrder *order.Order) {

	if targetOrder.IsExpired() {
		targetOrder.SetStatus(order.OrderStatusExpired)
		targetOrder.AddHistoryRecord("Срок хранения истек")
		fmt.Printf("Order %d can't be delivered because the storage has expired", targetOrder.ID)
	}

	if !targetOrder.IsRecievedByCourier() {
		fmt.Printf("Order %d must be recieved from courier", targetOrder.ID)
	}

	now := time.Now()
	targetOrder.SetDeliveredDate(&now)
	targetOrder.SetStatus(order.OrderStatusDelivered)
	targetOrder.AddHistoryRecord("Заказ выдан клиенту")

}

func (s *Pvz) GetAllRefunds() {
	orders := s.storage.GetOrders()

	var refundedOrders []*order.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	for _, order := range refundedOrders {
		fmt.Printf("%+v\n", *order)
	}
}

func (s *Pvz) GetHistory() {
	orders := s.storage.GetOrders()

	slices.SortFunc(orders, func(a *order.Order, b *order.Order) int {
		bHistory := b.History
		aHistory := a.History
		return bHistory[len(bHistory)-1].Timestamp.Compare(aHistory[len(aHistory)-1].Timestamp)
	})

	for _, order := range orders {
		fmt.Printf("%+v\n", *order)
	}
}
