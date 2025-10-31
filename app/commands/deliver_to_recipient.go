package commands

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
)
type ServeRecipientAction int
type OrderIdsSlice []int64

func (i *OrderIdsSlice) String() string {
    return fmt.Sprint(*i)
}

func (i *OrderIdsSlice) Set(value string) error {
   v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	*i = append(*i, v)
	return nil
}

var ErrInvalidCommand = errors.New("invalid command type")

const (
	Deliver ServeRecipientAction = iota
	AcceptReturn
)

var ServeRecipientActionName = map[ServeRecipientAction]string{
	Deliver:           "deliver",
	AcceptReturn:      "acceptReturn",
}

func (c ServeRecipientAction) String() string {
	return ServeRecipientActionName[c]
}

func ParseServeRecipientAction(value string) (ServeRecipientAction, error) {
	switch value {
	case Deliver.String():
		return Deliver, nil
	case AcceptReturn.String():
		return AcceptReturn, nil
	}
	
	return Deliver, fmt.Errorf("%q is not a valid command: %w", value, ErrInvalidCommand)
}

func ServeRecipient(args []string) (OrderIdsSlice, int64, ServeRecipientAction) {
	const argsCount = 2
	
	flagSet := flag.NewFlagSet("command", flag.ContinueOnError)
	
	var orderIds OrderIdsSlice
	
	flagSet.Var(&orderIds, "orders", "order ids")
	
	var (
		recipientId = flagSet.Int64("recipient", 0, "recipient id")
		action = flagSet.String("action", "", "recipient id")
	)
	
	parsedAction, ok := ParseServeRecipientAction(*action)
	
	if ok != nil {
		err := errors.New("ParseServeRecipientAction: incorrect error type")
		log.Fatal(err)
	}
	
	if nFlagSet := flagSet.NFlag(); nFlagSet != argsCount {
		err := errors.New("some arguments are missing")
		log.Fatal(err)
	}

	err := flagSet.Parse(args)
	
	if err != nil {
		log.Fatalf("flagSet.Parse: %s", err)
	}
	
	return orderIds, *recipientId, parsedAction
}