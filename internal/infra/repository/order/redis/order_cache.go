package cache_order_repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Staspol216/gh1/internal/config"
	"github.com/Staspol216/gh1/internal/domain/order"
)

type Cache struct {
	Rdb *redis.Client
}

func New(config *pvz_config.Config) *Cache {

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr(),
		Password: "",
		DB:       0,
	})

	cache := &Cache{
		Rdb: rdb,
	}

	return cache
}

func (cache *Cache) Healthcheck(ctx context.Context) error {
	_, err := cache.Rdb.Ping(ctx).Result()
	return err
}

func keyForOrder(id interface{}) string {
	return fmt.Sprintf("order:%v", id)
}

// GetOrder tries to get an order from redis and unmarshal it.
// Returns (*order.Order, nil) on hit, (nil, redis.Nil) on miss, or (nil, err) on error.
func (cache *Cache) GetOrder(ctx context.Context, id interface{}) (*pvz_domain.Order, error) {
	key := keyForOrder(id)
	b, err := cache.Rdb.Get(ctx, key).Bytes()
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
func (cache *Cache) SetOrder(ctx context.Context, order *pvz_domain.Order, ttl time.Duration) error {
	key := keyForOrder(order.ID)
	b, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return cache.Rdb.Set(ctx, key, b, ttl).Err()
}

func (cache *Cache) DeleteOrder(ctx context.Context, orderId int64) error {
	_, err := cache.Rdb.Del(ctx, keyForOrder(orderId)).Result()
	return err
}
