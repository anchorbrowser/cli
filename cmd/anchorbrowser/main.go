package main

import (
	"fmt"
	"os"

	"github.com/anchorbrowser/cli/internal/cli"
)

var version = "dev"

func main() {
	rootCmd, err := cli.NewRootCommand(version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
