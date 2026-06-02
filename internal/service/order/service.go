package pvz_order_service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/infrastructure/repository/order_outbox"
	"github.com/Staspol216/gh1/internal/ports"
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
}

type Pvz struct {
	outbox    order_outbox.OrderOutbox
	storage   pvz_domain.OrderStorager
	cache     OrdersCache
	txManager pvz_ports.TransactionManager
}

func New(storage pvz_domain.OrderStorager, outbox order_outbox.OrderOutbox, cache OrdersCache, txManager pvz_ports.TransactionManager) *Pvz {
	return &Pvz{
		outbox,
		storage,
		cache,
		txManager,
	}
}

func (s *Pvz) GetOrders(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {
	return s.storage.GetList(ctx, pagination)
}

func (s *Pvz) GetOrderByID(ctx context.Context, orderId int64) (*pvz_domain.Order, error) {
	order, err := s.cache.GetOrder(ctx, orderId)
	if err == nil {
		return order, nil
	}

	order, err = s.storage.GetByID(ctx, orderId)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetOrder(ctx, order, 0)

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
		if err := newOrder.ApplyPackaging(packagingType, additionalMembrana); err != nil {
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

			order, err := s.storage.GetByID(ctxTx, orderId)
			if err != nil {
				return err
			}

			if order.RecipientID != recipientId {
				return fmt.Errorf("order %d does not belong to recipient %d", orderId, recipientId)
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

	orders, err := s.storage.GetList(ctx, pagination)
	if err != nil {
		return nil, err
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

	orders, err := s.storage.GetList(ctx, pagination)
	if err != nil {
		return nil, err
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
