package warehouse

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/Staspol216/gh1/models/order"
	serveRecipient "github.com/Staspol216/gh1/models/serve_recipient_action"
)

type Warehouse struct {
	Orders []*order.Order `json:"orders"`
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

func (w *Warehouse) AcceptFromCourier(orderId int64, recipientId int64, expiration time.Time) {
	newOrder := order.New(orderId, recipientId, expiration)
	newOrder.SetStatus(order.OrderStatusReceived)
	newOrder.AddHistoryRecord("Заказ получен от курьера")
	w.saveOrder(newOrder)
}

func (w *Warehouse) DeleteExpiredOrderById(orderId int64) {
	defer func() {
		err := w.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	order, exists := w.findOrderById(orderId)

	if exists && order.IsExpired() {
		w.deleteOrderById(orderId)
	}
}

func (w *Warehouse) ServeRecipient(ordersIds []int64, recipientId int64, action string) {
	switch action {
	case serveRecipient.Deliver.String():
		w.deliverOrdersById(ordersIds, recipientId)
	case serveRecipient.Refund.String():
		w.refundOrdersById(ordersIds, recipientId)
	default:
		fmt.Println("Unknown action for ServeRecipient command")
	}
}

func (w *Warehouse) GetAllRefunds() {
	orders := w.getOrders()
	
	var refundedOrders []*order.Order
	
	for _, order := range orders {
		if order.IsRefunded() {
			refundedOrders = append(refundedOrders, order)
		}
	}
	
	for _, order := range refundedOrders {
		fmt.Printf("%+v\n", *order)
	}
}

func (w *Warehouse) GetHistory() {
	orders := w.getOrders()
	
	slices.SortFunc(orders, func(a *order.Order, b *order.Order) int {
		bHistory := b.History
		aHistory := a.History
		return bHistory[len(bHistory)-1].Timestamp.Compare(aHistory[len(aHistory)-1].Timestamp)
	})
	
	for _, order := range orders {
		fmt.Printf("%+v\n", *order)
	}
}

func (w *Warehouse) saveOrder(newOrder *order.Order) {
	_, exists := w.findOrderById(newOrder.ID)

	if exists {
		fmt.Println("Order already exists in the store")
		return
	}
	
	defer func() {
		err := w.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()

	w.addOrder(newOrder)
}

func (w *Warehouse) refundOrdersById(orderIds []int64, recipientId int64) *Warehouse {
	defer func() {
		err := w.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()
	
	orders := w.findRecipientOrdersByIds(orderIds, recipientId)
	
	for _, order := range orders {
		w.refundOrder(order)
	}
	
	return w
}

func (w *Warehouse) refundOrder(targetOrder *order.Order) *Warehouse {		
	if targetOrder.CanBeRefunded() {
		targetOrder.RefundByRecipient()
	} else {
		fmt.Printf("Order %d can not be refunded to recipient because refund time has expired", targetOrder.ID)
	}
	return w
}

func (w *Warehouse) deliverOrdersById(orderIds []int64, recipientId int64) *Warehouse {
	defer func() {
		err := w.saveStorageToFile()
		if err != nil {
			fmt.Println(err)
		}
	}()
	
	orders := w.findRecipientOrdersByIds(orderIds, recipientId)
	
	for _, order := range orders {
		w.deliverOrder(order)
	}
	
	return w
}

func (w *Warehouse) deliverOrder(targetOrder *order.Order) *Warehouse {
	
	if targetOrder.IsExpired() {
		targetOrder.SetStatus(order.OrderStatusExpired)
		targetOrder.AddHistoryRecord("Срок хранения истек")
		fmt.Printf("Order %d can't be delivered because the storage has expired", targetOrder.ID)
		return w
	}
	
	if !targetOrder.IsRecievedByCourier() {
		fmt.Printf("Order %d must be recieved from courier", targetOrder.ID)
		return w
	}
	
	now := time.Now()
	targetOrder.SetDeliveredDate(&now)
	targetOrder.SetStatus(order.OrderStatusDelivered)
	targetOrder.AddHistoryRecord("Заказ выдан клиенту")
	
	return w
}

func (w *Warehouse) deleteOrderById(orderId int64) *Warehouse {
	w.Orders = slices.DeleteFunc(w.Orders, func(o *order.Order) bool {
		return o.ID == orderId
	})
	
	return w
}

func (w *Warehouse) findOrderById(orderId int64) (*order.Order, bool) {
	for _, order := range w.Orders {
		if order.ID == orderId {
			return order, true
		}
	}

	return nil, false
}

func (w *Warehouse) getOrders() []*order.Order {
	return w.Orders
}

func (w *Warehouse) addOrder(order *order.Order) *Warehouse {
	w.Orders = append(w.Orders, order)
	return w
}

func (w *Warehouse) findRecipientOrdersByIds(orderIds []int64, recipientId int64) []*order.Order {
	var orders = make([]*order.Order, 0 , len(orderIds))
	
	for _, orderId := range orderIds {
		targetOrder, ok := w.findOrderById(orderId)
	
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
		log.Fatalln(encoderError)
	}

	fmt.Println("Struct was saved to json successfully")

	return nil
}