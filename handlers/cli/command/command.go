package command

type Command int

const (
	Unknown Command = iota
	Exit
	Help
	AcceptFromCourier
	ReturnToCourier
	ServeRecipient
	GetAllRefunds
	GetHistory
)

var CommandName = map[Command]string{
	Unknown:           "",
	Exit:              "exit",
	Help:              "help",
	AcceptFromCourier: "acceptFromCourier",
	ReturnToCourier:   "ReturnToCourier",
	ServeRecipient:    "serveRecipient",
	GetAllRefunds:     "getAllRefunds",
	GetHistory:        "getHistory",
}

func (c Command) String() string {
	return CommandName[c]
}
