package cli

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type OrderIds []int64

func (orderIds *OrderIds) String() string {
	var sb strings.Builder
	for idx, val := range *orderIds {
		if idx > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.FormatInt(val, 10))
	}
	return sb.String()
}

func (orderIds *OrderIds) Set(value string) error {
	for str := range strings.SplitSeq(value, ",") {
		val, err := strconv.ParseInt(strings.TrimSpace(str), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 value: %s", str)
		}
		*orderIds = append(*orderIds, val)
	}
	return nil
}

func ServeRecipient(args []string) (OrderIds, int64, string) {
	const argsCount = 3

	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)

	var orderIds OrderIds

	flagSet.Var(&orderIds, "orders", "orders ids")

	var (
		recipientId = flagSet.Int64("recipient", 0, "recipient id")
		action      = flagSet.String("action", "", "action")
	)

	err := flagSet.Parse(args)

	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}

	return orderIds, *recipientId, *action
}
