package command

type Command int

const (
	Unknown Command = iota
	Exit
	Help
	AcceptFromCourier
	ReturnFromCourier
)

var CommandName = map[Command]string{
	Unknown:           "",
	Exit:              "exit",
	Help:              "help",
	AcceptFromCourier: "acceptFromCourier",
	ReturnFromCourier: "returnFromCourier",
}

func (c Command) String() string {
	return CommandName[c]
}
