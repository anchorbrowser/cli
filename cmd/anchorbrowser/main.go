package main

import (
	"fmt"
	"os"

	"github.com/anchorbrowser/cli/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(version, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
