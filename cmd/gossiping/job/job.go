package job

import (
	"context"
	"github.com/f1shl3gs/gossiping/cmd/gossiping/internal"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "manage job",
	}

	cmd.PersistentFlags().String("host", "http://localhost:9000", "address of gossiping daemon to interact")

	cmd.AddCommand(add())
	cmd.AddCommand(edit())
	cmd.AddCommand(get())
	cmd.AddCommand(remove())
	cmd.AddCommand(list())

	return cmd
}

// gossiping job add name.json
func add() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			if name == "" {
				base := filepath.Base(filename)
				name = strings.TrimSuffix(base, filepath.Ext(base))
			}

			f, err := os.Open(args[0])
			if err != nil {
				return err
			}

			defer f.Close()

			cli := internal.ClientFromCmd(cmd)
			return cli.PostWithReader(context.Background(), "/jobs/"+name, f)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "name of the job")

	return cmd
}

func list() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list all jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := internal.ClientFromCmd(cmd)
			var jobs []string
			err := cli.Get(context.Background(), "/jobs", &jobs)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func edit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "edit a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}

func get() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get job by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}

func remove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "remove job by name",
		Aliases: []string{"rm"},
	}

	return cmd
}
