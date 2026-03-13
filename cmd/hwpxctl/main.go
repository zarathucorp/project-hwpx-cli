package main

import (
	"fmt"
	"os"

	"github.com/zarathu/project-hwpx-cli/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		type silencer interface {
			Silent() bool
		}
		type exitCoder interface {
			ExitCode() int
		}

		if err.Error() != "" {
			silent := false
			if quiet, ok := err.(silencer); ok {
				silent = quiet.Silent()
			}
			if !silent {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}

		var code int = 1
		if coded, ok := err.(exitCoder); ok {
			code = coded.ExitCode()
		}
		os.Exit(code)
	}
}
