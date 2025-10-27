package pvzApp

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Staspol216/gh1/app/commands"
	"github.com/Staspol216/gh1/command"
	order_storage "github.com/Staspol216/gh1/storage"
)

type App struct {}

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
		
		if (!ok) {
			log.Fatal("Cannot get command")
		}
		
		parsedCommand, err := command.Parse(commandString)
		
		if err != nil {
			log.Fatal("Incorrect command")
		}
		
		storage, strError := order_storage.New("storage/order_storage.json")
		
		if strError != nil {
			log.Fatal("order_storage.New: %w", strError)
		}
		
		fmt.Println(args)
		app.handleCommand(parsedCommand, args, storage)
		
	}
}

func (c *App) getCommandAndArgs(input string) (string, []string, bool) {
	fields := strings.Fields(input)
	
	fmt.Println(fields)
	
	if len(fields) == 0 {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}

func (c *App) handleCommand(v command.Command, args []string, storage *order_storage.OrderStorage) {
	switch v {
	case command.Exit:
		commands.Exit()
	case command.Help:
		commands.Help()
	case command.AcceptFromCourier:
		commands.AcceptFromCourier(args, storage)
	}
}