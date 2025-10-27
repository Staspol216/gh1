package models

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Staspol216/gh1/commands"
	"github.com/Staspol216/gh1/models/command"
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
		
		fmt.Println(args)
		app.handleCommand(parsedCommand, args)
		
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

func (c *App) handleCommand(v command.Command, args []string) {
	switch v {
	case command.Exit:
		commands.Exit()
	case command.Help:
		commands.Help()
	case command.AcceptFromCourier:
		commands.AcceptFromCourier(args)
	}
}