package main

import (
	"fmt"
	"os"
	"policy_exporter/exporter"
)

func main() {
	cmd := exporter.BuildCLI()
	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
