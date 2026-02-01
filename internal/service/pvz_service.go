package pvz_service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	pvz_model "github.com/Staspol216/gh1/internal/models/order"
	pvz_repository "github.com/Staspol216/gh1/internal/repository/storage"
	"github.com/Staspol216/gh1/internal/repository/tx_manager"
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
	storage   pvz_repository.Storager
	txManager tx_manager.TransactionManager
}

func New(storage pvz_repository.Storager, txManager tx_manager.TransactionManager) *Pvz {
	return &Pvz{
		storage,
		txManager,
	}
}

func (s *Pvz) GetOrders(ctx context.Context, pagination *pvz_model.Pagination) []*pvz_model.Order {
	return s.storage.GetList(ctx, pagination)
}

func (s *Pvz) GetOrderByID(ctx context.Context, orderId int64) (*pvz_model.Order, error) {
	order, err := s.storage.GetByID(ctx, orderId, "")

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *Pvz) GetOrdersByIDs(ctx context.Context, ordersIds []int64) ([]*pvz_model.Order, error) {
	orders, err := s.storage.GetByIDs(ctx, ordersIds)

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (p *Pvz) AcceptFromCourier(ctx context.Context, payload *pvz_model.OrderParams, packagingType string, additionalMembrana bool) (int64, error) {

	var orderId int64

	txError := p.txManager.RunReadCommitted(func(ctxTx context.Context) error {
		newOrder := pvz_model.New(payload)
		p.ApplyPackaging(newOrder, packagingType, additionalMembrana)
		newOrder.SetStatus(pvz_model.OrderStatusReceived)

		id, err := p.storage.Add(ctxTx, newOrder)
		if err != nil {
			orderId = 0
			return err
		}
		orderHistoryRecord := &pvz_model.OrderRecord{
			Timestamp:   time.Now(),
			Status:      newOrder.Status,
			Description: "Заказ получен от курьера",
		}
		p.storage.AddHistoryRecord(ctxTx, orderHistoryRecord, id)

		log.Println("order was succesfully added to the store")
		orderId = id

		return nil
	})

	return orderId, txError
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

func (p *Pvz) ReturnToCourier(ctx context.Context, orderId int64) error {

	txError := p.txManager.RunSerializable(func(ctxTx context.Context) error {
		order, err := p.storage.GetByID(ctxTx, orderId, "")

		if err != nil {
			return err
		}

		if !order.IsExpired() {
			return errors.New("order cannot be returned to courier as it's not expired")
		}

		p.storage.Delete(ctxTx, orderId)

		return nil
	})

	return txError
}

func (p *Pvz) ServeRecipient(ctx context.Context, ordersIds []int64, recipientId int64, action string) error {

	switch action {
	case Deliver.String():
		err := p.DeliverOrders(ctx, ordersIds, recipientId)
		if err != nil {
			return err
		}
	case Refund.String():
		err := p.RefundOrders(ctx, ordersIds, recipientId)
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown action for ServeRecipient command")
	}

	return nil
}

func (p *Pvz) RefundOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	txError := p.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

		recipientOrders, _ := p.storage.GetByRecipientId(ctxTx, recipientId, ordersIds)

		for _, order := range recipientOrders {

			if order.IsDelivered() && order.CanBeRefunded() {
				order.Refund()
				order.SetStatus(pvz_model.OrderStatusRefunded)
				newOrderRecord := &pvz_model.OrderRecord{
					Timestamp:   time.Now(),
					Status:      order.Status,
					Description: "Заказ возвращен от клиента",
				}
				p.storage.AddHistoryRecord(ctxTx, newOrderRecord, order.ID)
				err := p.storage.Update(ctxTx, order)
				return err
			} else {
				return fmt.Errorf("Order %d can not be refunded to recipient because refund time has expired or already refunded by recipient", order.ID)
			}
		}

		return nil
	})

	return txError
}

func (p *Pvz) DeliverOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	txError := p.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

		recipientOrders, _ := p.storage.GetByRecipientId(ctxTx, recipientId, ordersIds)

		for _, order := range recipientOrders {

			if !order.IsRecieved() {
				return fmt.Errorf("Order %d must be recieved from courier", order.ID)
			}

			if order.IsExpired() {
				order.SetStatus(pvz_model.OrderStatusExpired)
				newOrderRecord := &pvz_model.OrderRecord{
					Timestamp:   time.Now(),
					Status:      order.Status,
					Description: "Срок хранения истек",
				}
				p.storage.AddHistoryRecord(ctxTx, newOrderRecord, order.ID)
				err := p.storage.Update(ctxTx, order)
				return err
			} else {
				now := time.Now()
				order.SetDeliveredDate(&now)
				order.SetStatus(pvz_model.OrderStatusDelivered)
				newOrderRecord := &pvz_model.OrderRecord{
					Timestamp:   time.Now(),
					Status:      order.Status,
					Description: "Заказ выдан клиенту",
				}
				p.storage.AddHistoryRecord(ctxTx, newOrderRecord, order.ID)
				err := p.storage.Update(ctxTx, order)
				return err
			}
		}

		return nil
	})

	return txError
}

func (s *Pvz) GetAllRefunds(ctx context.Context, pagination *pvz_model.Pagination) []*pvz_model.Order {
	orders := s.storage.GetList(ctx, pagination)

	var refundedOrders []*pvz_model.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	return refundedOrders
}

func (s *Pvz) GetHistory(ctx context.Context, pagination *pvz_model.Pagination) []*pvz_model.Order {
	orders := s.storage.GetList(ctx, pagination)

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
