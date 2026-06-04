package pvz_order_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/infra/order_outbox"
	portsMocks "github.com/Staspol216/gh1/internal/ports/mocks"
	"github.com/Staspol216/gh1/internal/service/order/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testOrderID     int64 = 1
	testRecipientID int64 = 123
)

type pvzServiceTestFixture struct {
	service   *PvzService
	storage   *mocks.MockOrderStorage
	cache     *mocks.MockOrdersCache
	outbox    *mocks.MockOutbox
	txManager *portsMocks.MockTransactionManager
}

func newPvzServiceTestFixture(t *testing.T) *pvzServiceTestFixture {
	t.Helper()

	ctrl := gomock.NewController(t)

	storage := mocks.NewMockOrderStorage(ctrl)
	cache := mocks.NewMockOrdersCache(ctrl)
	outbox := mocks.NewMockOutbox(ctrl)
	txManager := portsMocks.NewMockTransactionManager(ctrl)

	return &pvzServiceTestFixture{
		service:   NewPvzService(storage, outbox, cache, txManager),
		storage:   storage,
		cache:     cache,
		outbox:    outbox,
		txManager: txManager,
	}
}

func newReceiveOrderParams() *pvz_domain.OrderParams {
	return &pvz_domain.OrderParams{
		RecipientId:    testRecipientID,
		ExpirationDate: time.Now().Add(24 * time.Hour),
		Weight:         5,
		Worth:          100,
	}
}

func newDeliveredTestOrder() *pvz_domain.Order {
	order := &pvz_domain.Order{
		ID:          testOrderID,
		RecipientID: testRecipientID,
	}
	order.Deliver()
	return order
}

func newReceivedTestOrder(expirationDate time.Time) *pvz_domain.Order {
	order := &pvz_domain.Order{
		ID:             testOrderID,
		RecipientID:    testRecipientID,
		ExpirationDate: expirationDate,
	}
	order.Received()
	return order
}

func newReceivedStoredTestOrder() *pvz_domain.Order {
	order := newReceivedTestOrder(time.Now().Add(24 * time.Hour))
	order.Worth = 121
	return order
}

func TestPvzService_ProcessOrderReceive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success process receive", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		storedOrder := newReceivedStoredTestOrder()

		fixture.storage.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, order *pvz_domain.Order) (int64, error) {
			assert.Equal(t, testRecipientID, order.RecipientID)
			assert.Equal(t, pvz_domain.OrderStatusReceived, order.Status)
			assert.Equal(t, float64(121), order.Worth)
			return testOrderID, nil
		})
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID)
		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(storedOrder, nil)
		fixture.outbox.EXPECT().AddTask(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, task *order_outbox.OrderOutboxTask) (int64, error) {
			assert.Equal(t, order_outbox.Created, task.Status)
			assert.Equal(t, pvz_domain.OrderStatusReceived, task.OrderStatus)
			assert.Equal(t, pvz_domain.OrderStatusDescription[pvz_domain.OrderStatusReceived], task.Description)
			return int64(1), nil
		})

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "box", true)

		// assert
		require.NoError(t, err)
		assert.Equal(t, storedOrder, order)
	})

	t.Run("returns error when packaging validation fails", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		payload.Weight = 10.01

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "bag", false)

		// assert
		require.Error(t, err)
		assert.Nil(t, order)
	})

	t.Run("returns error when add order fails", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		expectedErr := errors.New("add order failed")

		fixture.storage.EXPECT().Add(gomock.Any(), gomock.Any()).Return(int64(0), expectedErr)

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "box", false)

		// assert
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, order)
	})

	t.Run("returns error when add history record fails", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		expectedErr := errors.New("add history record failed")

		fixture.storage.EXPECT().Add(gomock.Any(), gomock.Any()).Return(testOrderID, nil)
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID).Return(int64(0), expectedErr)

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "box", false)

		// assert
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, order)
	})

	t.Run("returns error when get order fails", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		expectedErr := errors.New("get order failed")

		fixture.storage.EXPECT().Add(gomock.Any(), gomock.Any()).Return(testOrderID, nil)
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID)
		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(nil, expectedErr)

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "box", false)

		// assert
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, order)
	})

	t.Run("returns error when add outbox task fails", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		payload := newReceiveOrderParams()
		storedOrder := newReceivedStoredTestOrder()
		expectedErr := errors.New("add outbox task failed")

		fixture.storage.EXPECT().Add(gomock.Any(), gomock.Any()).Return(testOrderID, nil)
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID)
		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(storedOrder, nil)
		fixture.outbox.EXPECT().AddTask(gomock.Any(), gomock.Any()).Return(int64(0), expectedErr)

		// act
		order, err := fixture.service.ProcessOrderReceive(ctx, payload, "box", false)

		// assert
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, order)
	})
}

func TestPvzService_ProcessOrderRefund(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success process refund", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		order := newDeliveredTestOrder()

		fixture.storage.EXPECT().GetRecipientOrderByID(gomock.Any(), testOrderID, testRecipientID).Return(order, nil)
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), gomock.Any())
		fixture.storage.EXPECT().Update(gomock.Any(), gomock.Any())
		fixture.outbox.EXPECT().AddTask(gomock.Any(), gomock.Any())

		// act
		order, err := fixture.service.ProcessOrderRefund(ctx, testOrderID, testRecipientID)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testOrderID, order.ID)
		assert.Equal(t, pvz_domain.OrderStatusRefunded, order.Status)
	})
}

func TestPvzService_ProcessOrderDeliver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success process deliver", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		order := newReceivedTestOrder(time.Now().Add(time.Hour))

		fixture.storage.EXPECT().GetRecipientOrderByID(gomock.Any(), testOrderID, testRecipientID).Return(order, nil)
		fixture.storage.EXPECT().Update(gomock.Any(), gomock.Any())
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID)
		fixture.outbox.EXPECT().AddTask(gomock.Any(), gomock.Any())

		// act
		order, err := fixture.service.ProcessOrderDeliver(ctx, testOrderID, testRecipientID)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testOrderID, order.ID)
		assert.Equal(t, pvz_domain.OrderStatusDelivered, order.Status)
		assert.NotNil(t, order.DeliveredDate)
	})

	t.Run("expires order when storage date ended", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		order := newReceivedTestOrder(time.Now().Add(-time.Hour))

		fixture.storage.EXPECT().GetRecipientOrderByID(gomock.Any(), testOrderID, testRecipientID).Return(order, nil)
		fixture.storage.EXPECT().Update(gomock.Any(), gomock.Any())
		fixture.storage.EXPECT().AddHistoryRecord(gomock.Any(), gomock.Any(), testOrderID)
		fixture.outbox.EXPECT().AddTask(gomock.Any(), gomock.Any())

		// act
		order, err := fixture.service.ProcessOrderDeliver(ctx, testOrderID, testRecipientID)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testOrderID, order.ID)
		assert.Equal(t, pvz_domain.OrderStatusExpired, order.Status)
		assert.Nil(t, order.DeliveredDate)
	})

	t.Run("returns error when order is not received", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		order := newReceivedTestOrder(time.Now().Add(time.Hour))
		order.Deliver()

		fixture.storage.EXPECT().GetRecipientOrderByID(gomock.Any(), testOrderID, testRecipientID).Return(order, nil)

		// act
		order, err := fixture.service.ProcessOrderDeliver(ctx, testOrderID, testRecipientID)

		// assert
		require.Error(t, err)
		assert.Nil(t, order)
	})
}
