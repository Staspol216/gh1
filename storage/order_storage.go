package order_storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Staspol216/gh1/models/order"
)
type UserId = int64
type UserStore struct {
	orders []*order.Order
}
type OrderStorage struct {
	stores map[UserId]*UserStore
	path string
}

func New(path string) (*OrderStorage, error) {	
	b, err := os.ReadFile(path)
	
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}
	
	stores := make(map[UserId]*UserStore)

	err = json.Unmarshal(b, &stores)
	
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	
	newStorage := &OrderStorage{
		stores: stores,
		path: path,
	}
	
	return newStorage, nil
}

func (orderStorage *OrderStorage) GetUserStoreById(userId int64) *UserStore {
	return orderStorage.stores[userId]
}

func (cart *UserStore) AddOrder(order *order.Order) *UserStore {
	cart.orders = append(cart.orders, order)
	return cart
}