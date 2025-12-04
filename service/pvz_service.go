package Serivces

import (
	"errors"
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

func (s *Pvz) GetOrders() []*order.Order {
	return s.storage.GetList()
}

func (s *Pvz) AcceptFromCourier(payload *order.OrderParams, packagingType string, additionalMembrana bool) int64 {
	if isPast := utils.IsPastDate(payload.ExpirationDate); isPast {
		log.Println("expiration date can't be in the past")
		return 0
	}

	newOrder := order.New(payload)
	s.ApplyPackaging(newOrder, packagingType, additionalMembrana)
	newOrder.SetStatus(order.OrderStatusReceived)
	newHistoryRecord := &order.OrderRecord{
		Timestamp:   time.Now(),
		Status:      order.OrderStatusReceived,
		Description: "Заказ получен от курьера",
	}
	orderId, err := s.storage.Add(newOrder)
	if err != nil {
		return 0
	}
	s.storage.AddHistoryRecord(newHistoryRecord, orderId)

	log.Println("order was succesfully added to the store")

	return orderId
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
		log.Print("Unknown package type")
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

func (p *Pvz) ReturnToCourier(orderId int64) error {
	order, err := p.storage.GetByID(orderId)

	if err == nil && order.IsExpired() {
		p.storage.Delete(orderId)
	} else {
		return errors.New("order cannot be returned to courier as it's not expired")
	}
	return nil
}

func (p *Pvz) ServeRecipient(ordersIds []int64, recipientId int64, action string) error {

	recipientOrders, _ := p.storage.GetByRecipientId(recipientId)

	var targetOrders []*order.Order

	for _, order := range recipientOrders {
		if slices.Contains(ordersIds, order.ID) {
			targetOrders = append(targetOrders, order)
		}
	}

	switch action {
	case Deliver.String():
		p.DeliverOrders(targetOrders)
	case Refund.String():
		p.RefundOrders(targetOrders)
	default:
		return errors.New("unknown action for ServeRecipient command")
	}

	return nil
}

func (p *Pvz) RefundOrders(orders []*order.Order) {
	for _, order := range orders {
		p.RefundOrder(order)
		p.storage.Update(order)
	}
}

func (p *Pvz) RefundOrder(targetOrder *order.Order) {
	if targetOrder.IsDelivered() && targetOrder.CanBeRefunded() {
		targetOrder.Refund()
		targetOrder.SetStatus(order.OrderStatusRefunded)
		newOrderRecord := &order.OrderRecord{
			Timestamp:   time.Now(),
			Status:      order.OrderStatusRefunded,
			Description: "Заказ возвращен от клиента",
		}
		p.storage.AddHistoryRecord(newOrderRecord, targetOrder.ID)
	} else {
		log.Printf("Order %d can not be refunded to recipient because refund time has expired or already refunded by recipient", targetOrder.ID)
	}
}

func (p *Pvz) DeliverOrders(orders []*order.Order) {
	for _, order := range orders {
		p.DeliverOrder(order)
		p.storage.Update(order)
	}
}

func (p *Pvz) DeliverOrder(targetOrder *order.Order) {
	if targetOrder.IsExpired() {
		targetOrder.SetStatus(order.OrderStatusExpired)
		newOrderRecord := &order.OrderRecord{
			Timestamp:   time.Now(),
			Status:      order.OrderStatusExpired,
			Description: "Срок хранения истек",
		}
		p.storage.AddHistoryRecord(newOrderRecord, targetOrder.ID)
		log.Printf("Order %d can't be delivered because the storage has expired", targetOrder.ID)
	}

	if !targetOrder.IsRecieved() {
		log.Printf("Order %d must be recieved from courier", targetOrder.ID)
	}

	now := time.Now()
	targetOrder.SetDeliveredDate(&now)
	targetOrder.SetStatus(order.OrderStatusDelivered)
	newOrderRecord := &order.OrderRecord{
		Timestamp:   time.Now(),
		Status:      order.OrderStatusDelivered,
		Description: "Заказ выдан клиенту",
	}
	p.storage.AddHistoryRecord(newOrderRecord, targetOrder.ID)
}

func (s *Pvz) GetAllRefunds() []*order.Order {
	orders := s.storage.GetList()

	var refundedOrders []*order.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	return orders
}

func (s *Pvz) GetHistory() []*order.Order {
	orders := s.storage.GetList()

	slices.SortFunc(orders, func(a *order.Order, b *order.Order) int {
		var aT, bT time.Time

		if len(a.History) > 0 {
			aT = a.History[len(a.History)-1].Timestamp
		}
		if len(b.History) > 0 {
			bT = b.History[len(b.History)-1].Timestamp
		}

		return bT.Compare(aT)
	})

	return orders
}
