package commands

import (
	"fmt"

	"github.com/Staspol216/gh1/models/command"
)

func Help() {
	fmt.Println("Available commands:")
	for _, value := range command.CommandName {
		fmt.Println(value)
	}
}
