package cli

import (
	"flag"
	"log"
	"time"

	"github.com/Staspol216/gh1/utils"
)

const timeLayout = "2006-01-02"

type OrderPayload struct {
	OrderId          *int64
	RecipientId      *int64
	ExpirationDate   *time.Time
	Weight           *float64
	Worth            *float64
	Packaging        *string
	MembranaIncluded *bool
}

func AcceptFromCourier(args []string) *OrderPayload {
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)

	var (
		orderId          = flagSet.Int64("order", 0, "order id")
		recipientId      = flagSet.Int64("recipient", 0, "recipient id")
		expirationDate   = flagSet.String("expiration", "", "expiration datetime")
		weight           = flagSet.Float64("weight", 0, "weight")
		worth            = flagSet.Float64("worth", 0, "expiration worth")
		packaging        = flagSet.String("packaging", "", "packaging")
		membranaIncluded = flagSet.Bool("membranaIncluded", false, "membranaIncluded")
	)

	err := flagSet.Parse(args)

	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}

	required := []string{
		"order",
		"recipient",
		"expiration",
		"weight",
		"worth",
		"packaging",
	}

	provided := map[string]bool{}
	var missing []string

	flagSet.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	for _, name := range required {
		if !provided[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		log.Fatalf("missing required flags: %v", missing)
	}

	parsedExpirationDate, expirationErr := time.Parse(timeLayout, *expirationDate)

	if expirationErr != nil {
		log.Fatalf("time.Parse: %s", expirationErr)
	}

	if isPast := utils.IsPastDate(parsedExpirationDate); isPast {
		log.Fatal("expiration date can't be in the past")
	}

	return &OrderPayload{
		orderId,
		recipientId,
		&parsedExpirationDate,
		weight,
		worth,
		packaging,
		membranaIncluded,
	}
}
