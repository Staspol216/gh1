package inmemory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"

	pvz_model "github.com/Staspol216/gh1/internal/models/order"
)

type InMemoryOrderRepo struct {
	Orders []*pvz_model.Order `json:"orders"`
	path   string
}

func NewOrderRepo(path string) (*InMemoryOrderRepo, error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	orders := make([]*pvz_model.Order, 0)

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

func (p *InMemoryOrderRepo) GetAll(ctx context.Context) ([]*pvz_model.Order, error) {
	return p.Orders, nil
}

func (p *InMemoryOrderRepo) GetList(ctx context.Context, pagination *pvz_model.Pagination) ([]*pvz_model.Order, error) {
	return p.Orders, nil
}

func (p *InMemoryOrderRepo) Add(ctx context.Context, newOrder *pvz_model.Order) (int64, error) {
	_, err := p.GetByID(ctx, newOrder.ID)

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

func (r *InMemoryOrderRepo) AddHistoryRecord(ctx context.Context, record *pvz_model.OrderRecord, orderId int64) (int64, error) {
	return 0, nil
}

func (p *InMemoryOrderRepo) Update(ctx context.Context, updatedOrder *pvz_model.Order) error {
	err := p.saveStorageToFile()
	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (p *InMemoryOrderRepo) Delete(ctx context.Context, orderId int64) error {
	defer func() {
		err := p.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	p.Orders = slices.DeleteFunc(p.Orders, func(o *pvz_model.Order) bool {
		return o.ID == orderId
	})

	return nil
}

func (p *InMemoryOrderRepo) GetByID(ctx context.Context, orderId int64) (*pvz_model.Order, error) {
	for _, order := range p.Orders {
		if order.ID == orderId {
			return order, nil
		}
	}

	return nil, errors.New("order not found")
}

func (p *InMemoryOrderRepo) GetByIDs(ctx context.Context, orderIds []int64) ([]*pvz_model.Order, error) {

	var orders []*pvz_model.Order

	for _, order := range p.Orders {
		if slices.Contains(orderIds, order.ID) {
			orders = append(orders, order)
		}
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
