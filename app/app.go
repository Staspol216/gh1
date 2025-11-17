package pvzApp

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Staspol216/gh1/app/commands"
	"github.com/Staspol216/gh1/models/command"
)

type Pvz interface {
	AcceptFromCourier(orderId int64, recipientId int64, expiration time.Time)
	DeleteExpiredOrderById(id int64)
	ServeRecipient(orderIds []int64, recipientId int64, action string)
	GetAllRefunds()
	GetHistory()
}
type App struct {
	pvz Pvz
}

func New(pvz Pvz) *App {
	return &App{pvz}
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
		commands.Exit()
	case command.Help.String():
		commands.Help()
	case command.AcceptFromCourier.String():
		orderId, recipientId, parsedExpirationDate := commands.AcceptFromCourier(args)
		app.pvz.AcceptFromCourier(*orderId, *recipientId, *parsedExpirationDate)
	case command.ReturnFromCourier.String():
		orderId := commands.ReturnFromCourier(args)
		app.pvz.DeleteExpiredOrderById(orderId)
	case command.ServeRecipient.String():
		orderIds, recipientId, action := commands.ServeRecipient(args)
		app.pvz.ServeRecipient(orderIds, recipientId, action)
	case command.GetAllRefunds.String():
		app.pvz.GetAllRefunds()
	case command.GetHistory.String():
		app.pvz.GetHistory()
	default:
		commands.Unknown()
	}
}
