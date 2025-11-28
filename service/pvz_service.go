package Serivces

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/Staspol216/gh1/models/order"
	"github.com/Staspol216/gh1/storage"
	"github.com/Staspol216/gh1/utils"
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

func (s *Pvz) AcceptFromCourier(payload *order.OrderParams, packagingType string, additionalMembrana bool) {
	if isPast := utils.IsPastDate(payload.ExpirationDate); isPast {
		log.Println("expiration date can't be in the past")
		return
	}

	newOrder := order.New(payload)
	s.ApplyPackaging(newOrder, packagingType, additionalMembrana)
	newOrder.SetStatus(order.OrderStatusReceived)
	newOrder.AddHistoryRecord("Заказ получен от курьера")
	s.storage.Add(newOrder)
}

func (s *Pvz) getPackagingStrategy(packagingType string, additionalMembrana bool) PackagingStrategy {
	var Strategy PackagingStrategy

	switch packagingType {
	case "box":
		Strategy = &PackagingBoxStrategy{}
	case "bag":
		Strategy = &PackagingBagStrategy{}
	case "membrana":
		Strategy = &PackagingMembranaStrategy{}
	default:
		fmt.Print("Unknown package type")
	}

	if additionalMembrana && packagingType != "membrana" {
		Strategy = &MembranaDecorator{Strategy}
	}

	return Strategy
}

func (s *Pvz) ApplyPackaging(order *order.Order, packagingType string, additionalMembrana bool) error {
	packagingStrategy := s.getPackagingStrategy(packagingType, additionalMembrana)

	if err := packagingStrategy.Validate(order.Weight); err != nil {
		return err
	}

	order.Worth = packagingStrategy.CalculateWorth(order.Worth)
	return nil
}

func (p *Pvz) ReturnToCourier(orderId int64) {
	order, err := p.storage.GetByID(orderId)

	if err == nil && order.IsExpired() {
		p.storage.Delete(orderId)
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
	orders, _ := p.storage.GetByRecipientAndIds(recipientId, orderIds)

	for _, order := range orders {
		p.RefundOrder(order)
		p.storage.Update(order)
	}
}

func (p *Pvz) RefundOrder(targetOrder *order.Order) {
	if targetOrder.CanBeRefunded() {
		targetOrder.RefundByRecipient()
	} else {
		fmt.Printf("Order %d can not be refunded to recipient because refund time has expired", targetOrder.ID)
	}
}

func (p *Pvz) DeliverOrdersById(orderIds []int64, recipientId int64) {

	orders, _ := p.storage.GetByRecipientAndIds(recipientId, orderIds)

	for _, order := range orders {
		p.DeliverOrder(order)
		p.storage.Update(order)
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
	orders := s.storage.GetList()

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
	orders := s.storage.GetList()

	slices.SortFunc(orders, func(a *order.Order, b *order.Order) int {
		bHistory := b.History
		aHistory := a.History
		return bHistory[len(bHistory)-1].Timestamp.Compare(aHistory[len(aHistory)-1].Timestamp)
	})

	for _, order := range orders {
		fmt.Printf("%+v\n", *order)
	}
}
