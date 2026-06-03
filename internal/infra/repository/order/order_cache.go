package order

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Staspol216/gh1/internal/domain/order"
	"github.com/Staspol216/gh1/internal/ports"
)

type Cache struct {
	db pvz_ports.Cache
}

func NewOrderCache(db pvz_ports.Cache) *Cache {
	return &Cache{
		db: db,
	}
}

func (c *Cache) Healthcheck(ctx context.Context) error {
	_, err := c.db.Ping(ctx).Result()
	return err
}

func keyForOrder(id interface{}) string {
	return fmt.Sprintf("order:%v", id)
}

// GetOrder tries to get an order from redis and unmarshal it.
// Returns (*order.Order, nil) on hit, (nil, redis.Nil) on miss, or (nil, err) on error.
func (c *Cache) GetOrder(ctx context.Context, id interface{}) (*pvz_domain.Order, error) {
	key := keyForOrder(id)
	b, err := c.db.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var order pvz_domain.Order
	if err := json.Unmarshal(b, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// SetOrder stores an order in redis. ttl==0 means no expiration.
func (c *Cache) SetOrder(ctx context.Context, order *pvz_domain.Order, ttl time.Duration) error {
	key := keyForOrder(order.ID)
	b, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.db.Set(ctx, key, b, ttl).Err()
}

func (c *Cache) DeleteOrder(ctx context.Context, orderId int64) error {
	_, err := c.db.Del(ctx, keyForOrder(orderId)).Result()
	return err
}
