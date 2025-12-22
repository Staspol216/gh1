package pvz_cli

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Staspol216/gh1/internal/handlers/cli/command"
	pvz_model "github.com/Staspol216/gh1/internal/models/order"
	pvz_service "github.com/Staspol216/gh1/internal/service"
)

type CLIHandler struct {
	pvz *pvz_service.Pvz
}

func New(pvz *pvz_service.Pvz) *CLIHandler {
	return &CLIHandler{
		pvz,
	}
}

func (app *CLIHandler) Serve(context context.Context) error {
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

func (c *CLIHandler) getCommandAndArgs(input string) (string, []string, bool) {
	fields := strings.Fields(input)

	if len(fields) == 0 {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}

func (app *CLIHandler) handleCommand(v string, args []string) {
	pagination := &pvz_model.Pagination{
		Offset: 0,
		Limit:  1000,
	}
	switch v {
	case command.Exit.String():
		Exit()
	case command.Help.String():
		Help()
	case command.AcceptFromCourier.String():
		cliPayload := AcceptFromCourier(args)
		if cliPayload != nil {
			app.pvz.AcceptFromCourier(ToOrderParams(cliPayload), *cliPayload.Packaging, *cliPayload.MembranaIncluded)
		}
	case command.ReturnToCourier.String():
		orderId := ReturnToCourier(args)
		app.pvz.ReturnToCourier(orderId)
	case command.ServeRecipient.String():
		orderIds, recipientId, action := ServeRecipient(args)
		app.pvz.ServeRecipient(orderIds, recipientId, action)
	case command.GetAllRefunds.String():
		GetAllRefunds(args)
		app.pvz.GetAllRefunds(pagination)
	case command.GetHistory.String():
		GetHistory(args)
		app.pvz.GetHistory(pagination)
	default:
		Unknown()
	}
}

func ToOrderParams(p *OrderPayload) *pvz_model.OrderParams {
	return &pvz_model.OrderParams{
		OrderId:        *p.OrderId,
		RecipientId:    *p.RecipientId,
		ExpirationDate: *p.ExpirationDate,
		Weight:         *p.Weight,
		Worth:          *p.Worth,
	}
}
