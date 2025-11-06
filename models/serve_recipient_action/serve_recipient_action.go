package serveRecipient

import (
	"errors"
	"fmt"
)

type Action int

const (
	Deliver Action = iota
	Refund
)

var ServeRecipientActionName = map[Action]string{
	Deliver:     "deliver",
	Refund:      "refund",
}

func (c Action) String() string {
	return ServeRecipientActionName[c]
}

func ParseAction(value string) (Action, error) {
	switch value {
	case Deliver.String():
		return Deliver, nil
	case Refund.String():
		return Refund, nil
	}
	
	return Deliver, fmt.Errorf("%q is not a valid command: %w", value, errors.New("invalid command type"))
}