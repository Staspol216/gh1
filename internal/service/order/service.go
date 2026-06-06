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
	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/Staspol216/gh1/pkg/monitoring"
	"github.com/Staspol216/gh1/pkg/tracing"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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

func (s *PvzService) GetOrders(ctx context.Context, pagination *pvz_domain.Pagination) (orders []*pvz_domain.Order, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.GetOrders")
	span.SetTag("offset", pagination.Offset)
	span.SetTag("limit", pagination.Limit)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("get_orders", err)
	}()

	return s.storage.GetList(ctx, pagination)
}

func (s *PvzService) GetOrderByID(ctx context.Context, orderId int64, recipientId int64) (result *pvz_domain.Order, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.GetOrderByID")
	span.SetTag("order_id", orderId)
	span.SetTag("recipient_id", recipientId)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("get_order_by_id", err)
	}()

	order, err := s.cache.GetOrder(ctx, orderId)
	if err == nil {
		span.SetTag("cache", "hit")
		monitoring.ObserveCacheOperation("get_order_hit", nil)
		return order, nil
	}
	if errors.Is(err, redis.Nil) {
		span.SetTag("cache", "miss")
		monitoring.ObserveCacheOperation("get_order_miss", nil)
	} else {
		span.SetTag("cache", "error")
		monitoring.ObserveCacheOperation("get_order_error", err)
	}

	order, err = s.storage.GetRecipientOrderByID(ctx, orderId, recipientId)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.SetOrder(ctx, order, 0); cacheErr != nil {
		app_logger.MyLogger.Warn("failed to cache order after storage lookup",
			zap.Int64("order_id", order.ID),
			zap.Error(cacheErr),
		)
		monitoring.ObserveCacheOperation("set_order", cacheErr)
	} else {
		monitoring.ObserveCacheOperation("set_order", nil)
	}

	return order, nil
}

func (s *PvzService) GetOrdersByIDs(ctx context.Context, ordersIds []int64) (orders []*pvz_domain.Order, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.GetOrdersByIDs")
	span.SetTag("orders_count", len(ordersIds))
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("get_orders_by_ids", err)
	}()

	orders, err = s.storage.GetByIDs(ctx, ordersIds)

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *PvzService) AcceptFromCourier(ctx context.Context, payload *pvz_domain.OrderParams, packagingType string, additionalMembrana bool) (orderID *int64, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.AcceptFromCourier")
	span.SetTag("packaging_type", packagingType)
	span.SetTag("membrana_included", additionalMembrana)
	if payload != nil {
		span.SetTag("recipient_id", payload.RecipientId)
	}
	defer func() {
		if orderID != nil {
			span.SetTag("order_id", *orderID)
		}
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("accept_from_courier", err)
	}()

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
		monitoring.ObserveCacheOperation("set_order", err)
		return nil, err
	}
	monitoring.ObserveCacheOperation("set_order", nil)

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

func (s *PvzService) ReturnToCourier(ctx context.Context, orderId int64) (err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.ReturnToCourier")
	span.SetTag("order_id", orderId)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("return_to_courier", err)
	}()

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
			monitoring.ObserveCacheOperation("delete_order", err)
			return err
		}
		monitoring.ObserveCacheOperation("delete_order", nil)
	}

	return txError
}

func (s *PvzService) ServeRecipient(ctx context.Context, ordersIds []int64, recipientId int64, action string) (err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.ServeRecipient")
	span.SetTag("orders_count", len(ordersIds))
	span.SetTag("recipient_id", recipientId)
	span.SetTag("action", action)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("serve_recipient", err)
	}()

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

func (s *PvzService) RefundOrders(ctx context.Context, ordersIds []int64, recipientId int64) (err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.RefundOrders")
	span.SetTag("orders_count", len(ordersIds))
	span.SetTag("recipient_id", recipientId)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("refund_orders", err)
	}()

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
			monitoring.ObserveCacheOperation("set_order", err)
			return err
		}
		monitoring.ObserveCacheOperation("set_order", nil)
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

func (s *PvzService) DeliverOrders(ctx context.Context, ordersIds []int64, recipientId int64) (err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.DeliverOrders")
	span.SetTag("orders_count", len(ordersIds))
	span.SetTag("recipient_id", recipientId)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("deliver_orders", err)
	}()

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
			monitoring.ObserveCacheOperation("set_order", err)
			return err
		}
		monitoring.ObserveCacheOperation("set_order", nil)
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

func (s *PvzService) GetAllRefunds(ctx context.Context, pagination *pvz_domain.Pagination) (result []*pvz_domain.Order, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.GetAllRefunds")
	span.SetTag("offset", pagination.Offset)
	span.SetTag("limit", pagination.Limit)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("get_all_refunds", err)
	}()

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

func (s *PvzService) GetHistory(ctx context.Context, pagination *pvz_domain.Pagination) (result []*pvz_domain.Order, err error) {
	startTime := time.Now()
	span, ctx := tracing.StartSpanFromContext(ctx, "OrderService.GetHistory")
	span.SetTag("offset", pagination.Offset)
	span.SetTag("limit", pagination.Limit)
	defer func() {
		tracing.FinishSpan(span, startTime, err)
	}()
	defer func() {
		monitoring.ObserveOrderOperation("get_history", err)
	}()

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
