package pvz_service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/Staspol216/gh1/internal/db/tx_manager"
	pvz_model "github.com/Staspol216/gh1/internal/models/order"
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

type OrdersCache interface {
	GetOrder(ctx context.Context, id interface{}) (*pvz_model.Order, error)
	SetOrder(ctx context.Context, order *pvz_model.Order, ttl time.Duration) error
	DeleteOrder(ctx context.Context, orderId int64) error
	AddOrderToIndex(ctx context.Context, order *pvz_model.Order) error
	RemoveOrderFromIndex(ctx context.Context, orderID int64) error
	// GetOrders should return paginated orders. It may use the provided storage (repo)
	// to fetch data when cache misses or when an index is not available.
	GetList(ctx context.Context, pagination *pvz_model.Pagination, repo OrderRepo) ([]*pvz_model.Order, error)
	PopulateOrders(ctx context.Context, repo OrderRepo, ttl time.Duration) error
}

type Pvz struct {
	storage   Storager
	cache     OrdersCache
	txManager tx_manager.TransactionManager
}

func New(storage Storager, cache OrdersCache, txManager tx_manager.TransactionManager) *Pvz {
	return &Pvz{
		storage,
		cache,
		txManager,
	}
}

func (s *Pvz) GetOrders(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error) {

	orders, err := s.cache.GetList(ctx, pagination, s.storage)

	if err == nil {
		return orders, nil
	}

	return s.storage.GetList(ctx, pagination)
}

func (s *Pvz) GetOrderByID(ctx context.Context, orderId int64) (*pvz_model.Order, error) {
	orderCache, cacheErr := s.cache.GetOrder(ctx, orderId)

	if cacheErr != nil {
		return nil, cacheErr
	}

	if orderCache != nil {
		return orderCache, nil
	}

	order, err := s.storage.GetByID(ctx, orderId)

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

	var order *pvz_model.Order

	txError := p.txManager.RunReadCommitted(func(ctxTx context.Context) error {
		newOrder := pvz_model.New(payload)
		p.applyPackaging(newOrder, packagingType, additionalMembrana)
		newOrder.SetStatus(pvz_model.OrderStatusReceived)

		id, err := p.storage.Add(ctxTx, newOrder)
		if err != nil {
			return err
		}

		p.storage.AddHistoryRecord(ctxTx, pvz_model.NewOrderRecordReceived(), id)

		result, err := p.storage.GetByID(ctxTx, id)
		if err != nil {
			return err
		}

		order = result

		return nil
	})

	p.cache.SetOrder(ctx, order, 0)
	p.cache.AddOrderToIndex(ctx, order)

	return order.ID, txError
}

func (p *Pvz) ReturnToCourier(ctx context.Context, orderId int64) error {

	txError := p.txManager.RunRepeatableRead(func(ctxTx context.Context) error {
		order, err := p.storage.GetByID(ctxTx, orderId)

		if err != nil {
			return err
		}

		if !order.IsExpired() {
			return errors.New("order cannot be returned to courier as it's not expired")
		}

		p.storage.Delete(ctxTx, orderId)

		return nil
	})

	if txError == nil {
		p.cache.RemoveOrderFromIndex(ctx, orderId)
		p.cache.DeleteOrder(ctx, orderId)
	}

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

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_model.Order

		txError := p.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, _ := p.storage.GetByID(ctx, orderId)

			if order.RecipientID != recipientId {
				return fmt.Errorf("Order %d does not belong to recipient %d", orderId, recipientId)
			}

			if order == nil {
				return fmt.Errorf("Order %d not found", orderId)
			}

			if order.IsDelivered() && order.CanBeRefunded() {
				order.Refund()
				p.storage.AddHistoryRecord(ctxTx, pvz_model.NewOrderRecordRefunded(), order.ID)
				err := p.storage.Update(ctxTx, order)
				updatedOrder = order
				return err
			} else {
				return fmt.Errorf("Order %d can not be refunded to recipient because refund time has expired or already refunded by recipient", order.ID)
			}

		})

		if txError == nil {
			p.cache.SetOrder(ctx, updatedOrder, 0)
		}

		if txError != nil {
			return txError
		}
	}

	return nil
}

func (p *Pvz) DeliverOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_model.Order

		txError := p.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, _ := p.storage.GetByID(ctx, orderId)

			if order.RecipientID != recipientId {
				return fmt.Errorf("Order %d does not belong to recipient %d", orderId, recipientId)
			}

			if order == nil {
				return fmt.Errorf("Order %d not found", orderId)
			}

			if !order.IsRecieved() {
				return fmt.Errorf("Order %d must be recieved from courier", order.ID)
			}

			if order.IsExpired() {
				order.Expire()
				err := p.storage.Update(ctxTx, order)
				p.storage.AddHistoryRecord(ctxTx, pvz_model.NewOrderRecordExpired(), order.ID)
				updatedOrder = order
				return err
			} else {
				order.Deliver()
				err := p.storage.Update(ctxTx, order)
				p.storage.AddHistoryRecord(ctxTx, pvz_model.NewOrderRecordDelivered(), order.ID)
				updatedOrder = order
				return err
			}
		})

		if txError == nil {
			p.cache.SetOrder(ctx, updatedOrder, 0)
		}

		if txError != nil {
			return txError
		}
	}

	return nil
}

func (p *Pvz) GetAllRefunds(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error) {

	orders, err := p.cache.GetList(ctx, pagination, p.storage)
	if err != nil {
		orders, err = p.storage.GetList(ctx, pagination)
		if err != nil {
			return nil, err
		}
	}

	var refundedOrders []*pvz_model.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	return refundedOrders, nil
}

func (p *Pvz) GetHistory(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error) {

	orders, err := p.cache.GetList(ctx, pagination, p.storage)
	if err != nil {
		orders, err = p.storage.GetList(ctx, pagination)
		if err != nil {
			return nil, err
		}
	}

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

	return orders, nil
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

func (s *Pvz) applyPackaging(order *pvz_model.Order, packagingType string, additionalMembrana bool) error {
	packagingStrategy := s.getPackagingStrategy(packagingType, additionalMembrana)

	if err := packagingStrategy.Validate(order.Weight); err != nil {
		return err
	}

	order.Worth = packagingStrategy.CalculateWorth(order.Worth)
	return nil
}
