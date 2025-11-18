package inmemory

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/Staspol216/gh1/models/order"
)

type InMemoryOrderStorage struct {
	Orders []*order.Order `json:"orders"`
	path   string
}

func New(path string) (*InMemoryOrderStorage, error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	orders := make([]*order.Order, 0)

	err = json.Unmarshal(b, &orders)

	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	newStorage := &InMemoryOrderStorage{
		Orders: orders,
		path:   path,
	}

	return newStorage, nil
}

func (p *InMemoryOrderStorage) GetOrders() []*order.Order {
	return p.Orders
}

func (p *InMemoryOrderStorage) addOrder(order *order.Order) *InMemoryOrderStorage {
	p.Orders = append(p.Orders, order)
	return p
}

func (p *InMemoryOrderStorage) SaveOrder(newOrder *order.Order) {
	_, exists := p.FindOrderById(newOrder.ID)

	if exists {
		fmt.Println("Order already exists in the store")
		return
	}

	defer func() {
		err := p.SaveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	p.addOrder(newOrder)
}

func (p *InMemoryOrderStorage) DeleteOrderById(orderId int64) {
	p.Orders = slices.DeleteFunc(p.Orders, func(o *order.Order) bool {
		return o.ID == orderId
	})
}

func (p *InMemoryOrderStorage) FindOrderById(orderId int64) (*order.Order, bool) {
	for _, order := range p.Orders {
		if order.ID == orderId {
			return order, true
		}
	}

	return nil, false
}

func (p *InMemoryOrderStorage) FindRecipientOrdersByIds(orderIds []int64, recipientId int64) []*order.Order {
	var orders = make([]*order.Order, 0, len(orderIds))

	for _, orderId := range orderIds {
		targetOrder, ok := p.FindOrderById(orderId)

		if !ok {
			fmt.Printf("Order %d not founded in storage", orderId)
			continue
		}

		if targetOrder.RecipientID != recipientId {
			fmt.Printf("Order %d does not belong to the buyer %d, it can't be delivered", orderId, recipientId)
			continue
		}

		orders = append(orders, targetOrder)
	}

	return orders
}

func (p *InMemoryOrderStorage) SaveStorageToFile() error {
	f, err := os.OpenFile(p.path, os.O_RDWR|os.O_TRUNC, 0666)

	if err != nil {
		log.Fatalln(err, 1)
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "\t")

	encoderError := encoder.Encode(p.Orders)
	if encoderError != nil {
		log.Fatalln(encoderError)
	}

	fmt.Println("Struct was saved to json successfully")

	return nil
}
