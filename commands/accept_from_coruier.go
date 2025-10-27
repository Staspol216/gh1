package commands

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"time"
)

const argsCount = 3
const timeLayout = "2006-01-02"

func AcceptFromCourier(args []string) {
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)
	
	var (
		order = flagSet.Int64("order", 0, "order id")
		recipient = flagSet.Int64("recipient", 0, "recipient id")
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
		err := errors.New("expiration date can't be in the past")
		log.Fatal(err) 
	}
	
	fmt.Println(*order, *recipient, parsedExpirationDate)
	
	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}
	
	fmt.Println(args)	
}

func isPastDate(date time.Time) bool {
	nowDate := time.Now()
	res := nowDate.Compare(date)
	
	return res == 1
}