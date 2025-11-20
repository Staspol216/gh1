package pvzApp

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Staspol216/gh1/handlers/cli"
	"github.com/Staspol216/gh1/handlers/cli/command"
	"github.com/Staspol216/gh1/models/order"
)

type Pvz interface {
	AcceptFromCourier(payload *order.OrderParams, packagingType string, additionalMembrana bool)
	ReturnToCourier(id int64)
	ServeRecipient(orderIds []int64, recipientId int64, action string)
	GetAllRefunds()
	GetHistory()
}
type App struct {
	pvz Pvz
}

func New(pvz Pvz) *App {
	return &App{
		pvz,
	}
}

func (app *App) Run() {
	for {
		fmt.Printf("> ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()

		command, args, ok := app.getCommandAndArgs(input)

		if !ok {
			log.Fatal("Cannot get command")
		}

		app.handleCommand(command, args)
	}
}

func (c *App) getCommandAndArgs(input string) (string, []string, bool) {
	fields := strings.Fields(input)

	if len(fields) == 0 {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}

func (app *App) handleCommand(v string, args []string) {
	switch v {
	case command.Exit.String():
		cli.Exit()
	case command.Help.String():
		cli.Help()
	case command.AcceptFromCourier.String():
		cliPayload := cli.AcceptFromCourier(args)
		app.pvz.AcceptFromCourier(ToOrderParams(cliPayload), *cliPayload.Packaging, *cliPayload.MembranaIncluded)
	case command.ReturnFromCourier.String():
		orderId := cli.ReturnFromCourier(args)
		app.pvz.ReturnToCourier(orderId)
	case command.ServeRecipient.String():
		orderIds, recipientId, action := cli.ServeRecipient(args)
		app.pvz.ServeRecipient(orderIds, recipientId, action)
	case command.GetAllRefunds.String():
		cli.GetAllRefunds(args)
		app.pvz.GetAllRefunds()
	case command.GetHistory.String():
		cli.GetHistory(args)
		app.pvz.GetHistory()
	default:
		cli.Unknown()
	}
}

func ToOrderParams(p *cli.OrderPayload) *order.OrderParams {
	return &order.OrderParams{
		OrderId:        *p.OrderId,
		RecipientId:    *p.RecipientId,
		ExpirationDate: *p.ExpirationDate,
		Weight:         *p.Weight,
		Worth:          *p.Worth,
	}
}
