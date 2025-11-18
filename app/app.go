package pvzApp

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Staspol216/gh1/cli"
	"github.com/Staspol216/gh1/cli/command"
)

type Pvz interface {
	AcceptFromCourier(orderId int64, recipientId int64, expiration time.Time)
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
		orderId, recipientId, parsedExpirationDate := cli.AcceptFromCourier(args)
		app.pvz.AcceptFromCourier(*orderId, *recipientId, *parsedExpirationDate)
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
