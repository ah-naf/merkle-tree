package main

import (
	"fmt"
	"os"

	"github.com/ah-naf/merkle-cli/internals/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
