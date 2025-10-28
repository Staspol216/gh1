package commands

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Staspol216/gh1/models/order"
	order_storage "github.com/Staspol216/gh1/storage"
)

const argsCount = 3
const timeLayout = "2006-01-02"

func AcceptFromCourier(args []string, storage *order_storage.OrderStorage) *order.Order {
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)
	
	var (
		orderId = flagSet.Int64("order", 0, "order id")
		recipientId = flagSet.Int64("recipient", 0, "recipient id")
		expiration = flagSet.String("expiration", "", "expiration datetime")
	)
	
	err := flagSet.Parse(args)
	
	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguemts are missing")
		log.Fatal(err)
	}
	
	parsedExpirationDate, expirationErr := time.Parse(timeLayout, *expiration)
	
	if expirationErr != nil {
		log.Fatalf("time.Parse: %s", expirationErr)
	}

	if isPast := isPastDate(parsedExpirationDate); isPast {
		fmt.Printf("expiration date can't be in the past")
	}
	
	fmt.Println(*orderId, *recipientId, parsedExpirationDate)
	
	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}
	
	newOrder := order.New(*orderId, parsedExpirationDate)
	
	fmt.Println(newOrder, 32233232)
	
	storage.SaveOrderToStore(*recipientId, newOrder)
	
	
	return newOrder
}

func isPastDate(date time.Time) bool {
	nowDate := time.Now()
	res := nowDate.Compare(date)
	
	return res == 1
}