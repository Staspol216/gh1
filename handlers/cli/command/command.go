package command

type Command int

const (
	Unknown Command = iota
	Exit
	Help
	AcceptFromCourier
	ReturnFromCourier
	ServeRecipient
	GetAllRefunds
	GetHistory
)

var CommandName = map[Command]string{
	Unknown:           "",
	Exit:              "exit",
	Help:              "help",
	AcceptFromCourier: "acceptFromCourier",
	ReturnFromCourier: "returnFromCourier",
	ServeRecipient:    "serveRecipient",
	GetAllRefunds:     "getAllRefunds",
	GetHistory:        "getHistory",
}

func (c Command) String() string {
	return CommandName[c]
}
