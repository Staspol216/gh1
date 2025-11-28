package inmemory

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/Staspol216/gh1/models/order"
)

type InMemoryOrderRepo struct {
	Orders []*order.Order `json:"orders"`
	path   string
}

func NewOrderRepo(path string) (*InMemoryOrderRepo, error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	orders := make([]*order.Order, 0)

	err = json.Unmarshal(b, &orders)

	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	newStorage := &InMemoryOrderRepo{
		Orders: orders,
		path:   path,
	}

	return newStorage, nil
}

func (p *InMemoryOrderRepo) GetList() []*order.Order {
	return p.Orders
}

func (p *InMemoryOrderRepo) Add(newOrder *order.Order) (int64, error) {
	_, err := p.GetByID(newOrder.ID)

	if err == nil {
		return 0, errors.New("order already exists in the store")
	}

	defer func() {
		err := p.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	p.Orders = append(p.Orders, newOrder)

	return newOrder.ID, nil
}

func (p *InMemoryOrderRepo) Update(updatedOrder *order.Order) {
	err := p.saveStorageToFile()
	if err != nil {
		fmt.Println(err)
	}
}

func (p *InMemoryOrderRepo) Delete(orderId int64) error {
	defer func() {
		err := p.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	p.Orders = slices.DeleteFunc(p.Orders, func(o *order.Order) bool {
		return o.ID == orderId
	})

	return nil
}

func (p *InMemoryOrderRepo) GetByID(orderId int64) (*order.Order, error) {
	for _, order := range p.Orders {
		if order.ID == orderId {
			return order, nil
		}
	}

	return nil, errors.New("order not found")
}

func (p *InMemoryOrderRepo) GetByRecipientAndIds(recipientId int64, orderIds []int64) ([]*order.Order, error) {
	var orders = make([]*order.Order, 0, len(orderIds))

	for _, orderId := range orderIds {
		targetOrder, err := p.GetByID(orderId)

		if err != nil {
			fmt.Printf("Order %d not founded in storage", orderId)
			continue
		}

		if targetOrder.RecipientID != recipientId {
			fmt.Printf("Order %d does not belong to the buyer %d, it can't be delivered", orderId, recipientId)
			continue
		}

		orders = append(orders, targetOrder)
	}

	return orders, nil
}

func (p *InMemoryOrderRepo) saveStorageToFile() error {
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
