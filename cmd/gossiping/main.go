package main

import (
	"os"

	"github.com/f1shl3gs/gossiping/cmd/gossiping/cluster"
	"github.com/f1shl3gs/gossiping/cmd/gossiping/job"
	"github.com/f1shl3gs/gossiping/cmd/gossiping/serve"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "gossiping",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(serve.Serve())
	rootCmd.AddCommand(cluster.New())
	rootCmd.AddCommand(job.New())
	rootCmd.AddCommand(autoComplete())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func autoComplete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-complete",
		Short: "generate auto complete script for bash/fish/zsh",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Parent()
			output := os.Stdout

			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(output)
			case "fish":
				return rootCmd.GenFishCompletion(output, true)
			case "zsh":
				return rootCmd.GenZshCompletion(output)
			default:
				return errors.Errorf("unexpected type %q", args[0])
			}
		},
	}

	return cmd
}
