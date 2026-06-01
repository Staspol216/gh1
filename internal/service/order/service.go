package pvz_order_service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
	psql_order_outbox_repo "github.com/Staspol216/gh1/internal/infrastructure/repository/order_outbox"
	pvz_ports "github.com/Staspol216/gh1/internal/ports"
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
	GetOrder(ctx context.Context, id interface{}) (*pvz_domain.Order, error)
	SetOrder(ctx context.Context, order *pvz_domain.Order, ttl time.Duration) error
	DeleteOrder(ctx context.Context, orderId int64) error
	AddOrderToIndex(ctx context.Context, order *pvz_domain.Order) error
	RemoveOrderFromIndex(ctx context.Context, orderID int64) error
	// GetList should return paginated orders. It may use the provided storage (repo)
	// to fetch data when cache misses or when an index is not available.
	GetList(ctx context.Context, pagination *pvz_domain.Pagination, repo pvz_domain.OrderStorager) ([]*pvz_domain.Order, error)
}

type Pvz struct {
	outbox    psql_order_outbox_repo.OrderOutboxRepo
	storage   pvz_domain.OrderStorager
	cache     OrdersCache
	txManager pvz_ports.TransactionManager
}

func New(storage pvz_domain.OrderStorager, outbox psql_order_outbox_repo.OrderOutboxRepo, cache OrdersCache, txManager pvz_ports.TransactionManager) *Pvz {
	return &Pvz{
		outbox,
		storage,
		cache,
		txManager,
	}
}

func (s *Pvz) GetOrders(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {

	orders, err := s.cache.GetList(ctx, pagination, s.storage)

	if err == nil {
		return orders, nil
	}

	return s.storage.GetList(ctx, pagination)
}

func (s *Pvz) GetOrderByID(ctx context.Context, orderId int64) (*pvz_domain.Order, error) {
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

func (s *Pvz) GetOrdersByIDs(ctx context.Context, ordersIds []int64) ([]*pvz_domain.Order, error) {
	orders, err := s.storage.GetByIDs(ctx, ordersIds)

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *Pvz) AcceptFromCourier(ctx context.Context, payload *pvz_domain.OrderParams, packagingType string, additionalMembrana bool) (*int64, error) {

	var order *pvz_domain.Order

	txError := s.txManager.RunReadCommitted(func(ctxTx context.Context) error {
		newOrder := pvz_domain.New(payload)
		if err := s.applyPackaging(newOrder, packagingType, additionalMembrana); err != nil {
			return err
		}

		newOrder.SetStatus(pvz_domain.OrderStatusReceived)

		id, err := s.storage.Add(ctxTx, newOrder)
		if err != nil {
			return err
		}

		orderRecord := pvz_domain.NewOrderRecordReceived()

		if _, er := s.storage.AddHistoryRecord(ctxTx, orderRecord, id); er != nil {
			return er
		}

		result, err := s.storage.GetByID(ctxTx, id)
		if err != nil {
			return err
		}

		task := &pvz_domain.OrderOutboxTask{
			Status:      pvz_domain.Created,
			CreatedAt:   time.Now(),
			OrderStatus: orderRecord.Status,
			Description: orderRecord.Description,
			Timestamp:   orderRecord.Timestamp,
		}

		_, outboxErr := s.outbox.AddTask(ctxTx, task)
		if outboxErr != nil {
			return outboxErr
		}

		order = result

		return nil
	})

	if txError != nil {
		return nil, txError
	}

	if err := s.cache.SetOrder(ctx, order, 0); err != nil {
		return nil, err
	}

	if err := s.cache.AddOrderToIndex(ctx, order); err != nil {
		return nil, err
	}

	return &order.ID, txError
}

func (s *Pvz) ReturnToCourier(ctx context.Context, orderId int64) error {

	txError := s.txManager.RunRepeatableRead(func(ctxTx context.Context) error {
		order, err := s.storage.GetByID(ctxTx, orderId)

		if err != nil {
			return err
		}

		if !order.IsExpired() {
			return errors.New("order cannot be returned to courier as it's not expired")
		}

		if errDel := s.storage.Delete(ctxTx, orderId); errDel != nil {
			return errDel
		}

		return nil
	})

	if txError == nil {
		if err := s.cache.RemoveOrderFromIndex(ctx, orderId); err != nil {
			return err
		}
		if err := s.cache.DeleteOrder(ctx, orderId); err != nil {
			return err
		}
	}

	return txError
}

func (s *Pvz) ServeRecipient(ctx context.Context, ordersIds []int64, recipientId int64, action string) error {

	switch action {
	case Deliver.String():
		err := s.DeliverOrders(ctx, ordersIds, recipientId)
		if err != nil {
			return err
		}
	case Refund.String():
		err := s.RefundOrders(ctx, ordersIds, recipientId)
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown action for ServeRecipient command")
	}

	return nil
}

func (s *Pvz) RefundOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_domain.Order

		txError := s.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, _ := s.storage.GetByID(ctxTx, orderId)

			if order.RecipientID != recipientId {
				return fmt.Errorf("order %d does not belong to recipient %d", orderId, recipientId)
			}

			if order == nil {
				return fmt.Errorf("order %d not found", orderId)
			}

			if order.IsDelivered() && order.CanBeRefunded() {
				order.Refund()

				orderRecord := pvz_domain.NewOrderRecordRefunded()
				if _, err := s.storage.AddHistoryRecord(ctxTx, orderRecord, order.ID); err != nil {
					return err
				}

				if err := s.storage.Update(ctxTx, order); err != nil {
					return err
				}

				job := &pvz_domain.OrderOutboxTask{
					Status:      pvz_domain.Created,
					CreatedAt:   time.Now(),
					OrderStatus: orderRecord.Status,
					Description: orderRecord.Description,
					Timestamp:   orderRecord.Timestamp,
				}

				_, outboxErr := s.outbox.AddTask(ctxTx, job)
				if outboxErr != nil {
					return outboxErr
				}

				updatedOrder = order
			} else {
				return fmt.Errorf(`order %d can not be refunded to recipient because refund time has expired or it has already refunded by recipient`, order.ID)
			}

			return nil
		})

		if txError != nil {
			return txError
		}

		if err := s.cache.SetOrder(ctx, updatedOrder, 0); err != nil {
			return err
		}
	}

	return nil
}

func (s *Pvz) DeliverOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_domain.Order

		txError := s.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, _ := s.storage.GetByID(ctxTx, orderId)

			if order.RecipientID != recipientId {
				return fmt.Errorf("order %d does not belong to recipient %d", orderId, recipientId)
			}

			if order == nil {
				return fmt.Errorf("order %d not found", orderId)
			}

			if !order.IsReceived() {
				return fmt.Errorf("order %d must be received from courier", order.ID)
			}

			if order.IsExpired() {
				order.Expire()
				if err := s.storage.Update(ctxTx, order); err != nil {
					return err
				}

				orderRecord := pvz_domain.NewOrderRecordExpired()

				_, err := s.storage.AddHistoryRecord(ctxTx, orderRecord, order.ID)
				if err != nil {
					return err
				}

				task := &pvz_domain.OrderOutboxTask{
					Status:      pvz_domain.Created,
					CreatedAt:   time.Now(),
					OrderStatus: orderRecord.Status,
					Description: orderRecord.Description,
					Timestamp:   orderRecord.Timestamp,
				}

				_, outboxErr := s.outbox.AddTask(ctxTx, task)
				if outboxErr != nil {
					return outboxErr
				}

				updatedOrder = order
			} else {
				order.Deliver()
				if err := s.storage.Update(ctxTx, order); err != nil {
					return err
				}

				orderRecord := pvz_domain.NewOrderRecordDelivered()

				if _, err := s.storage.AddHistoryRecord(ctxTx, orderRecord, order.ID); err != nil {
					return err
				}

				task := &pvz_domain.OrderOutboxTask{
					Status:      pvz_domain.Created,
					CreatedAt:   time.Now(),
					OrderStatus: orderRecord.Status,
					Description: orderRecord.Description,
					Timestamp:   orderRecord.Timestamp,
				}

				_, outboxErr := s.outbox.AddTask(ctxTx, task)
				if outboxErr != nil {
					return outboxErr
				}

				updatedOrder = order
			}

			return nil
		})

		if txError != nil {
			return txError
		}

		if err := s.cache.SetOrder(ctx, updatedOrder, 0); err != nil {
			return err
		}
	}

	return nil
}

func (s *Pvz) GetAllRefunds(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {

	orders, err := s.cache.GetList(ctx, pagination, s.storage)
	if err != nil {
		orders, err = s.storage.GetList(ctx, pagination)
		if err != nil {
			return nil, err
		}
	}

	var refundedOrders []*pvz_domain.Order

	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}

	return refundedOrders, nil
}

func (s *Pvz) GetHistory(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {

	orders, err := s.cache.GetList(ctx, pagination, s.storage)

	if err != nil {
		orders, err = s.storage.GetList(ctx, pagination)
		if err != nil {
			return nil, err
		}
	}

	slices.SortFunc(orders, func(a *pvz_domain.Order, b *pvz_domain.Order) int {
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

func (s *Pvz) getPackagingStrategy(packagingType string) pvz_domain.PackagingStrategy {
	switch packagingType {
	case "box":
		return &pvz_domain.PackagingBoxStrategy{}
	case "bag":
		return &pvz_domain.PackagingBagStrategy{}
	case "membrana":
		return &pvz_domain.PackagingMembranaStrategy{}
	default:
		return &pvz_domain.PackagingBoxStrategy{}
	}
}

func (s *Pvz) applyPackaging(order *pvz_domain.Order, packagingType string, additionalMembrana bool) error {
	packagingStrategy := s.getPackagingStrategy(packagingType)

	if additionalMembrana && packagingType != "membrana" {
		packagingStrategy = &pvz_domain.MembranaDecorator{Strategy: packagingStrategy}
	}

	if err := packagingStrategy.Validate(order.Weight); err != nil {
		return err
	}

	order.Worth = packagingStrategy.CalculateWorth(order.Worth)
	return nil
}
