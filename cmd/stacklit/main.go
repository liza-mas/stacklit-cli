package main

import (
	"fmt"
	"os"

	"github.com/glincker/stacklit/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		code, reportErr := cli.ErrorToExit(err)
		if reportErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", reportErr)
		}
		os.Exit(code)
	}
}
