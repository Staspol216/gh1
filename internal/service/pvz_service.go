package pvz_service

import (
	"errors"
	"log"
	"slices"
	"time"

	pvz_model "github.com/Staspol216/gh1/internal/models/order"
	pvz_repository "github.com/Staspol216/gh1/internal/repository/storage"
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
	storage pvz_repository.Storager
}

func New(storage pvz_repository.Storager) *Pvz {
	return &Pvz{
		storage,
	}
}

func (s *Pvz) GetOrders(pagination *pvz_model.Pagination) []*pvz_model.Order {
	return s.storage.GetList(pagination)
}

func (s *Pvz) GetOrderByID(orderId int64) (*pvz_model.Order, error) {
	order, err := s.storage.GetByID(orderId)

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *Pvz) AcceptFromCourier(payload *pvz_model.OrderParams, packagingType string, additionalMembrana bool) int64 {
	newOrder := pvz_model.New(payload)
	s.ApplyPackaging(newOrder, packagingType, additionalMembrana)
	newOrder.SetStatus(pvz_model.OrderStatusReceived)
	newHistoryRecord := &pvz_model.OrderRecord{
		Timestamp:   time.Now(),
		Status:      pvz_model.OrderStatusReceived,
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

func (s *Pvz) ApplyPackaging(order *pvz_model.Order, packagingType string, additionalMembrana bool) error {
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

	var targetOrders []*pvz_model.Order

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

func (p *Pvz) RefundOrders(orders []*pvz_model.Order) {
	for _, order := range orders {
		p.RefundOrder(order)
		p.storage.Update(order)
	}
}

func (p *Pvz) RefundOrder(targetOrder *pvz_model.Order) {
	if targetOrder.IsDelivered() && targetOrder.CanBeRefunded() {
		targetOrder.Refund()
		targetOrder.SetStatus(pvz_model.OrderStatusRefunded)
		newOrderRecord := &pvz_model.OrderRecord{
			Timestamp:   time.Now(),
			Status:      pvz_model.OrderStatusRefunded,
			Description: "Заказ возвращен от клиента",
		}
		p.storage.AddHistoryRecord(newOrderRecord, targetOrder.ID)
	} else {
		log.Printf("Order %d can not be refunded to recipient because refund time has expired or already refunded by recipient", targetOrder.ID)
	}
}

func (p *Pvz) DeliverOrders(orders []*pvz_model.Order) {
	for _, order := range orders {
		p.DeliverOrder(order)
		p.storage.Update(order)
	}
}

func (p *Pvz) DeliverOrder(targetOrder *pvz_model.Order) {
	if targetOrder.IsExpired() {
		targetOrder.SetStatus(pvz_model.OrderStatusExpired)
		newOrderRecord := &pvz_model.OrderRecord{
			Timestamp:   time.Now(),
			Status:      pvz_model.OrderStatusExpired,
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
	targetOrder.SetStatus(pvz_model.OrderStatusDelivered)
	newOrderRecord := &pvz_model.OrderRecord{
		Timestamp:   time.Now(),
		Status:      pvz_model.OrderStatusDelivered,
		Description: "Заказ выдан клиенту",
	}
	p.storage.AddHistoryRecord(newOrderRecord, targetOrder.ID)
}

func (s *Pvz) GetAllRefunds(pagination *pvz_model.Pagination) []*pvz_model.Order {
	orders := s.storage.GetList(pagination)

	var refundedOrders []*pvz_model.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	return refundedOrders
}

func (s *Pvz) GetHistory(pagination *pvz_model.Pagination) []*pvz_model.Order {
	orders := s.storage.GetList(pagination)

	slices.SortFunc(orders, func(a *pvz_model.Order, b *pvz_model.Order) int {
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
