package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spothero/tools/cli"
)

func main() {
	b := false
	cmd := cobra.Command{
		Use:              "whatever",
		PersistentPreRun: cli.CobraBindEnvironmentVariables("whatever"),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("it's: %v!\n", b)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&b, "bool", true, "bool")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
