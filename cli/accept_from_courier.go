package cli

import (
	"errors"
	"flag"
	"log"
	"time"

	"github.com/Staspol216/gh1/utils"
)

const timeLayout = "2006-01-02"

func AcceptFromCourier(args []string) (*int64, *int64, *time.Time) {
	const argsCount = 3
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)

	var (
		orderId        = flagSet.Int64("order", 0, "order id")
		recipientId    = flagSet.Int64("recipient", 0, "recipient id")
		expirationDate = flagSet.String("expiration", "", "expiration datetime")
	)

	err := flagSet.Parse(args)

	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}

	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	parsedExpirationDate, expirationErr := time.Parse(timeLayout, *expirationDate)

	if expirationErr != nil {
		log.Printf("time.Parse: %s", expirationErr)
		return nil, nil, nil
	}

	if isPast := utils.IsPastDate(parsedExpirationDate); isPast {
		log.Fatal("expiration date can't be in the past")
	}

	return orderId, recipientId, &parsedExpirationDate
}
