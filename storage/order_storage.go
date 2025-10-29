package order_storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/Staspol216/gh1/models/order"
)
type UserId = int64
type UserStore struct {
	Orders []*order.Order `json:"order"`
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

func (s *OrderStorage) saveStorageToFile() error {
	f, err := os.OpenFile(s.path, os.O_RDWR|os.O_TRUNC, 0666)
	
	if err != nil {
		log.Fatalln(err, 1)
	}
	
	defer f.Close()
	
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "\t")
	
	encoderError := encoder.Encode(s.stores)
	if encoderError != nil {
		fmt.Println(encoderError)
		log.Fatalln(encoderError, 2)
	}
	
	fmt.Println("Struct was saved to json successfully")
	
	return nil
}


func (storage *OrderStorage) GetUserStoreById(userId int64) *UserStore {
	return storage.stores[userId]
}

func (storage *OrderStorage) SaveOrderToStore(recipientId int64, newOrder *order.Order) bool {
	defer func() {
		err := storage.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()
	
	store := storage.GetUserStoreById(recipientId)
	
	if store == nil {
		storage.stores[recipientId] = &UserStore{
			Orders: make([]*order.Order, 0),
		}
		store = storage.stores[recipientId]
	}
	
	_, exists := storage.stores[recipientId].FindOrderById(newOrder.ID)
	
	if exists {
		fmt.Println("Order already exists in the store")
		return false
	}
	
	fmt.Println(newOrder)
	store.AddOrder(newOrder)
	return true
}

func (storage *OrderStorage) DeleteOrderFromStore(recipientId int64, orderId int64) *UserStore  {
	store := storage.GetUserStoreById(recipientId)
	order, exists := store.FindOrderById(orderId)
	
	if exists && order.IsExpiredOrder() {
		store.DeleteOrderById(orderId)
	}
	
	return store
}

func (store *UserStore) DeleteOrderById(orderId int64)  {	
	store.Orders = slices.DeleteFunc(store.Orders, func(o *order.Order) bool {
		return o.ID == orderId
	})
}

func (store *UserStore) FindOrderById(orderId int64) (*order.Order, bool) {
	for _, order := range store.Orders {
		if order.ID == orderId {
			return order, true
		}
	}
	
	return nil, false
}

func (store *UserStore) AddOrder(order *order.Order) *UserStore {
	store.Orders = append(store.Orders, order)
	return store
}