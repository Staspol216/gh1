package warehouse

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/Staspol216/gh1/models/order"
)

type UserId = int64

type Warehouse struct {
	Orders []*order.Order `json:"order"`
	path   string
}

func New(path string) (*Warehouse, error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	orders := make([]*order.Order, 0)

	err = json.Unmarshal(b, &orders)

	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	newStorage := &Warehouse{
		Orders: orders,
		path:   path,
	}

	return newStorage, nil
}

const argsCount = 3
const timeLayout = "2006-01-02"


func (w *Warehouse) saveStorageToFile() error {
	f, err := os.OpenFile(w.path, os.O_RDWR|os.O_TRUNC, 0666)

	if err != nil {
		log.Fatalln(err, 1)
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "\t")

	encoderError := encoder.Encode(w.Orders)
	if encoderError != nil {
		fmt.Println(encoderError)
		log.Fatalln(encoderError, 2)
	}

	fmt.Println("Struct was saved to json successfully")

	return nil
}

func (w *Warehouse) SaveOrder(newOrder *order.Order) bool {
	defer func() {
		err := w.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	_, exists := w.FindOrderById(newOrder.ID)

	if exists {
		fmt.Println("Order already exists in the store")
		return false
	}

	w.AddOrder(newOrder)
	return true
}

func (w *Warehouse) DeleteExpiredOrderById(orderId int64) *Warehouse {

	order, exists := w.FindOrderById(orderId)

	if exists && order.IsExpired() {
		w.DeleteOrderById(orderId)
	}

	return w
}

func (w *Warehouse) DeleteOrderById(orderId int64) {
	w.Orders = slices.DeleteFunc(w.Orders, func(o *order.Order) bool {
		return o.ID == orderId
	})
}

func (w *Warehouse) FindOrderById(orderId int64) (*order.Order, bool) {
	for _, order := range w.Orders {
		if order.ID == orderId {
			return order, true
		}
	}

	return nil, false
}

func (w *Warehouse) AddOrder(order *order.Order) *Warehouse {
	w.Orders = append(w.Orders, order)
	return w
}
