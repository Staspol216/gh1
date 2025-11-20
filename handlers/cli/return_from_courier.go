package cli

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

	err := flagSet.Parse(args)

	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}

	return *orderId
}
