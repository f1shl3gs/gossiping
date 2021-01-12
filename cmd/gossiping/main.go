package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "gossiping",
		RunE:         launch,
		SilenceUsage: true,
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
