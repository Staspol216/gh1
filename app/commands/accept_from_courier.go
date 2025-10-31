package commands

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Staspol216/gh1/models/order"
	"github.com/Staspol216/gh1/utils"
)

const timeLayout = "2006-01-02"

func AcceptFromCourier(args []string) *order.Order {
	const argsCount = 3
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)

	var (
		orderId     = flagSet.Int64("order", 0, "order id")
		recipientId = flagSet.Int64("recipient", 0, "recipient id")
		expiration  = flagSet.String("expiration", "", "expiration datetime")
	)

	err := flagSet.Parse(args)
	
	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}

	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	parsedExpirationDate, expirationErr := time.Parse(timeLayout, *expiration)

	if expirationErr != nil {
		fmt.Printf("time.Parse: %s", expirationErr)
		return nil
	}

	if isPast := utils.IsPastDate(parsedExpirationDate); isPast {
		fmt.Println("expiration date can't be in the past")
		return nil
	}

	newOrder := order.New(*orderId, *recipientId, parsedExpirationDate)

	return newOrder
}