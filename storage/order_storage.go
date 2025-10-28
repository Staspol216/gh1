package order_storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Staspol216/gh1/models/order"
)
type UserId = int64
type UserStore struct {
	Orders []*order.Order
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
	
	fmt.Println(s.stores[2].Orders[0], 3423423423424324242)
	encoderError := encoder.Encode(s.stores)
	if encoderError != nil {
		fmt.Println(encoderError)
		log.Fatalln(encoderError, 2)
	}
	
	fmt.Println("Struct successfully written to json")
	
	return nil
}


func (orderStorage *OrderStorage) GetUserStoreById(userId int64) *UserStore {
	return orderStorage.stores[userId]
}

func (store *UserStore) FindOrderById(orderId int64) (*order.Order, bool) {
	for _, order := range store.Orders {
		if order.ID == orderId {
			return order, true
		}
	}
	
	return nil, false
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


func (store *UserStore) AddOrder(order *order.Order) *UserStore {
	fmt.Println(store, order, "store, order")
	store.Orders = append(store.Orders, order)
	return store
}