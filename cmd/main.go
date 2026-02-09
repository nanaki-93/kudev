package main

import (
	"fmt"
	"os"

	"github.com/nanaki-93/kudev/cmd/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
