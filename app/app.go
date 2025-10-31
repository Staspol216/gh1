package pvzApp

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Staspol216/gh1/app/commands"
	"github.com/Staspol216/gh1/models/command"
	warehouse "github.com/Staspol216/gh1/storage"
)

type App struct{}

func New() *App {
	return &App{}
}

func (app *App) Run() {
	for {
		fmt.Printf("> ")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()

		commandString, args, ok := app.getCommandAndArgs(input)

		if !ok {
			log.Fatal("Cannot get command")
		}

		warehouse, strError := warehouse.New("storage/warehouse.json")

		if strError != nil {
			log.Fatal("warehouse.New: %w", strError)
		}

		fmt.Println(args)
		app.handleCommand(commandString, args, warehouse)

	}
}

func (c *App) getCommandAndArgs(input string) (string, []string, bool) {
	fields := strings.Fields(input)

	if len(fields) == 0 {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}

func (c *App) handleCommand(v string, args []string, w *warehouse.Warehouse) {
	switch v {
	case command.Exit.String():
		commands.Exit()
	case command.Help.String():
		commands.Help()
	case command.AcceptFromCourier.String():
		order := commands.AcceptFromCourier(args)
		w.SaveOrder(order)
	case command.ReturnFromCourier.String():
		orderId := commands.ReturnFromCourier(args)
		w.DeleteExpiredOrderById(orderId)
	default:
		commands.Unknown()
	}
}
