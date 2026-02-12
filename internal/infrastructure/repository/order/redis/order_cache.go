package cache_order_repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	pvz_domain "github.com/Staspol216/gh1/internal/domain/order"
)

type Cache struct {
	Rdb *redis.Client
}

func New() *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
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

// GetOrderFromCache tries to get an order from redis and unmarshal it.
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

// SetOrderInCache stores an order in redis. ttl==0 means no expiration.
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

// PopulateOrders loads all orders from storage and writes them to redis using a pipeline.
// Also builds an index ZSET (orders:idx) for efficient paginated range queries.
// It honors the provided ctx (should have timeout or be derived from signal context).
func (cache *Cache) PopulateOrders(ctx context.Context, repo pvz_domain.OrderStorager, ttl time.Duration) error {
	orders, err := repo.GetAll(ctx)
	if err != nil {
		return err
	}

	// Pipeline to SET individual order data and build ZSET index
	pipe := cache.Rdb.Pipeline()
	for _, order := range orders {
		b, err := json.Marshal(order)
		if err != nil {
			return err
		}
		// SET full order data at order:{id}
		pipe.Set(ctx, keyForOrder(order.ID), b, ttl)
		pipe.ZAdd(ctx, "orders:idx", redis.Z{Score: float64(order.ID), Member: order.ID})
	}
	_, err = pipe.Exec(ctx)
	return err
}

// GetList returns paginated orders using Redis ZSET index (orders:idx).
// It fetches order IDs via ZRANGE, then pipelined GETs. Cache misses are filled
// from the repo and cached for future use.
func (cache *Cache) GetList(ctx context.Context, pagination *pvz_domain.Pagination, repo pvz_domain.OrderStorager) ([]*pvz_domain.Order, error) {
	start := int64(pagination.Offset)
	end := int64(pagination.Offset + pagination.Limit - 1)

	idxs, err := cache.Rdb.ZRange(ctx, "orders:idx", start, end).Result()

	if err != nil {
		if err == redis.Nil {
			return []*pvz_domain.Order{}, nil
		}
		return nil, err
	}

	if len(idxs) == 0 {
		return []*pvz_domain.Order{}, nil
	}

	keys := make([]string, len(idxs))
	ids := make([]int64, len(idxs))

	for i, id := range idxs {
		keys[i] = keyForOrder(id)
		id, _ := strconv.ParseInt(id, 10, 64)
		ids[i] = id
	}

	values, err := cache.Rdb.MGet(ctx, keys...).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make([]*pvz_domain.Order, len(idxs))
	missingPositionById := make(map[int64][]int)

	// Build list
	for i, v := range values {
		if v == nil {
			id := ids[i]
			missingPositionById[id] = append(missingPositionById[id], i)
			continue
		}
		value, ok := v.(string)
		if !ok {
			id := ids[i]
			missingPositionById[id] = append(missingPositionById[id], i)
			continue
		}
		var order pvz_domain.Order
		if err := json.Unmarshal([]byte(value), &order); err != nil {
			id := ids[i]
			missingPositionById[id] = append(missingPositionById[id], i)
			continue
		}
		result[i] = &order
	}

	if len(missingPositionById) > 0 {

		// Get from DB
		missingIDs := make([]int64, 0, len(missingPositionById))
		for id := range missingPositionById {
			missingIDs = append(missingIDs, id)
		}
		fromDB, err := repo.GetByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}

		pipe := cache.Rdb.Pipeline()
		for _, order := range fromDB {
			// Fill missing orders from DB
			for _, pos := range missingPositionById[order.ID] {
				result[pos] = order
			}
			// Set cache value
			b, _ := json.Marshal(order)
			pipe.Set(ctx, keyForOrder(order.ID), b, 0)
			pipe.ZAdd(ctx, "orders:idx", redis.Z{Score: float64(order.ID), Member: order.ID})
		}
		_, _ = pipe.Exec(ctx)
	}

	return result, nil
}

// AddOrderToIndex adds an order to the ZSET index. Call this after creating a new order.
// Score is the order ID for stable ordering.
func (cache *Cache) AddOrderToIndex(ctx context.Context, order *pvz_domain.Order) error {
	return cache.Rdb.ZAdd(ctx, "orders:idx", redis.Z{
		Score:  float64(order.ID),
		Member: order.ID,
	}).Err()
}

// RemoveOrderFromIndex removes an order from the ZSET index. Call this after deleting an order.
func (cache *Cache) RemoveOrderFromIndex(ctx context.Context, orderID int64) error {
	return cache.Rdb.ZRem(ctx, "orders:idx", orderID).Err()
}
