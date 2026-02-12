package main

import (
	"os"

	"github.com/nanaki-93/kudev/cmd/commands"
)

func main() {
	exitCode := commands.Execute()
	os.Exit(exitCode)
}
