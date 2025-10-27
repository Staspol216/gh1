package command

import (
	"errors"
	"fmt"
)

var ErrInvalidCommand = errors.New("invalid command")

type Command int

const (
	Unknown Command = iota
	Exit
	Help
	AcceptFromCourier
)

var commandName = map[Command]string{
	Unknown: 				"",
    Exit:   				"exit",
    Help: 					"help",
	AcceptFromCourier: 		"acceptFromCourier",
}

func (c Command) String() string {
    return commandName[c]
}

func Parse(value string) (Command, error) {
	switch value {
	case Exit.String():
		return Exit, nil
	case Help.String():
		return Help, nil
	case AcceptFromCourier.String():
		return AcceptFromCourier, nil
	}
	
	return Unknown, fmt.Errorf("%q is not a valid command: %w", value, ErrInvalidCommand)
}