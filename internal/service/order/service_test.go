package pvz_order_service

import (
	"context"
	"testing"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
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

func TestPvzService_ProcessOrderRefund(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success process refund", func(t *testing.T) {
		t.Parallel()
		// arrange
		fixture := newPvzServiceTestFixture(t)
		order := newDeliveredTestOrder()

		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(order, nil)
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

		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(order, nil)
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

		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(order, nil)
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

		fixture.storage.EXPECT().GetByID(gomock.Any(), testOrderID).Return(order, nil)

		// act
		order, err := fixture.service.ProcessOrderDeliver(ctx, testOrderID, testRecipientID)

		// assert
		require.Error(t, err)
		assert.Nil(t, order)
	})
}
