package commands

import (
	"errors"
	"flag"
	"log"
)

func ReturnFromCourier(args []string) int64 {
	const argsCount = 1
	
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)

	var (
		orderId = flagSet.Int64("order", 0, "order id")
	)
	
	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	err := flagSet.Parse(args)
	
	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}
	
	return *orderId
}