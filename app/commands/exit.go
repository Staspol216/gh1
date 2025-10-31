package commands

import (
	"fmt"
	"os"
)

func Exit() {
	fmt.Println("Exit")
	os.Exit(0)
}
