package pvz_order_service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	"github.com/Staspol216/gh1/internal/ports"
)

type PvzService struct {
	outbox    Outbox
	storage   OrderStorage
	cache     OrdersCache
	txManager pvz_ports.TransactionManager
}

func NewPvzService(
	storage OrderStorage,
	outbox Outbox,
	cache OrdersCache,
	txManager pvz_ports.TransactionManager,
) *PvzService {
	return &PvzService{
		outbox,
		storage,
		cache,
		txManager,
	}
}

func (s *PvzService) GetOrders(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {
	return s.storage.GetList(ctx, pagination)
}

func (s *PvzService) GetOrderByID(ctx context.Context, orderId int64, recipientId int64) (*pvz_domain.Order, error) {
	order, err := s.cache.GetOrder(ctx, orderId)
	if err == nil {
		return order, nil
	}

	order, err = s.storage.GetRecipientOrderByID(ctx, orderId, recipientId)
	if err != nil {
		return nil, err
	}

	_ = s.cache.SetOrder(ctx, order, 0)

	return order, nil
}

func (s *PvzService) GetOrdersByIDs(ctx context.Context, ordersIds []int64) ([]*pvz_domain.Order, error) {
	orders, err := s.storage.GetByIDs(ctx, ordersIds)

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *PvzService) AcceptFromCourier(ctx context.Context, payload *pvz_domain.OrderParams, packagingType string, additionalMembrana bool) (*int64, error) {

	var order *pvz_domain.Order

	txError := s.txManager.RunReadCommitted(func(ctxTx context.Context) error {
		result, err := s.ProcessOrderReceive(ctxTx, payload, packagingType, additionalMembrana)
		if err != nil {
			return err
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

func (s *PvzService) ProcessOrderReceive(ctxTx context.Context, payload *pvz_domain.OrderParams, packagingType string, additionalMembrana bool) (*pvz_domain.Order, error) {
	newOrder := pvz_domain.NewOrder(payload)
	if err := newOrder.ApplyPackaging(packagingType, additionalMembrana); err != nil {
		return nil, err
	}

	newOrder.Received()

	id, err := s.storage.Add(ctxTx, newOrder)
	if err != nil {
		return nil, err
	}

	orderRecord := pvz_domain.NewOrderRecordReceived()

	if _, er := s.storage.AddHistoryRecord(ctxTx, orderRecord, id); er != nil {
		return nil, er
	}

	result, err := s.storage.GetByID(ctxTx, id)
	if err != nil {
		return nil, err
	}

	task := &order_outbox.OrderOutboxTask{
		Status:      order_outbox.Created,
		CreatedAt:   time.Now(),
		OrderStatus: orderRecord.Status,
		Description: orderRecord.Description,
		Timestamp:   orderRecord.Timestamp,
	}

	_, outboxErr := s.outbox.AddTask(ctxTx, task)
	if outboxErr != nil {
		return nil, outboxErr
	}
	return result, nil
}

func (s *PvzService) ReturnToCourier(ctx context.Context, orderId int64) error {

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

func (s *PvzService) ServeRecipient(ctx context.Context, ordersIds []int64, recipientId int64, action string) error {

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

func (s *PvzService) RefundOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_domain.Order

		txError := s.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, err := s.ProcessOrderRefund(ctxTx, orderId, recipientId)
			if err != nil {
				return err
			}

			updatedOrder = order

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

func (s *PvzService) ProcessOrderRefund(ctx context.Context, orderId int64, recipientId int64) (*pvz_domain.Order, error) {
	order, err := s.storage.GetRecipientOrderByID(ctx, orderId, recipientId)
	if err != nil {
		return nil, err
	}

	if !order.CanBeRefunded() {
		return nil, fmt.Errorf(`order %d can not be refunded to recipient because refund time has expired or it has already refunded by recipient`, order.ID)
	}

	order.Refund()

	orderRecord := pvz_domain.NewOrderRecordRefunded()
	if _, err := s.storage.AddHistoryRecord(ctx, orderRecord, order.ID); err != nil {
		return nil, err
	}

	if err := s.storage.Update(ctx, order); err != nil {
		return nil, err
	}

	job := &order_outbox.OrderOutboxTask{
		Status:      order_outbox.Created,
		CreatedAt:   time.Now(),
		OrderStatus: orderRecord.Status,
		Description: orderRecord.Description,
		Timestamp:   orderRecord.Timestamp,
	}

	_, outboxErr := s.outbox.AddTask(ctx, job)
	if outboxErr != nil {
		return nil, outboxErr
	}
	return order, nil
}

func (s *PvzService) DeliverOrders(ctx context.Context, ordersIds []int64, recipientId int64) error {

	for _, orderId := range ordersIds {

		var updatedOrder *pvz_domain.Order

		txError := s.txManager.RunRepeatableRead(func(ctxTx context.Context) error {

			order, err2 := s.ProcessOrderDeliver(ctxTx, orderId, recipientId)
			if err2 != nil {
				return err2
			}

			updatedOrder = order

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

func (s *PvzService) ProcessOrderDeliver(ctxTx context.Context, orderId int64, recipientId int64) (*pvz_domain.Order, error) {
	order, err := s.storage.GetRecipientOrderByID(ctxTx, orderId, recipientId)
	if err != nil {
		return nil, err
	}

	if !order.IsReceived() {
		return nil, fmt.Errorf("order %d must be received from courier", order.ID)
	}

	if order.IsExpired() {
		order.Expire()
		if err := s.storage.Update(ctxTx, order); err != nil {
			return nil, err
		}

		orderRecord := pvz_domain.NewOrderRecordExpired()

		_, err := s.storage.AddHistoryRecord(ctxTx, orderRecord, order.ID)
		if err != nil {
			return nil, err
		}

		task := &order_outbox.OrderOutboxTask{
			Status:      order_outbox.Created,
			CreatedAt:   time.Now(),
			OrderStatus: orderRecord.Status,
			Description: orderRecord.Description,
			Timestamp:   orderRecord.Timestamp,
		}

		_, outboxErr := s.outbox.AddTask(ctxTx, task)
		if outboxErr != nil {
			return nil, outboxErr
		}

		return order, nil
	}

	order.Deliver()
	if err := s.storage.Update(ctxTx, order); err != nil {
		return nil, err
	}

	orderRecord := pvz_domain.NewOrderRecordDelivered()

	if _, err := s.storage.AddHistoryRecord(ctxTx, orderRecord, order.ID); err != nil {
		return nil, err
	}

	task := &order_outbox.OrderOutboxTask{
		Status:      order_outbox.Created,
		CreatedAt:   time.Now(),
		OrderStatus: orderRecord.Status,
		Description: orderRecord.Description,
		Timestamp:   orderRecord.Timestamp,
	}

	_, outboxErr := s.outbox.AddTask(ctxTx, task)
	if outboxErr != nil {
		return nil, outboxErr
	}

	return order, nil
}

func (s *PvzService) GetAllRefunds(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {

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

func (s *PvzService) GetHistory(ctx context.Context, pagination *pvz_domain.Pagination) ([]*pvz_domain.Order, error) {

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
