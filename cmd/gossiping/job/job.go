package job

import (
	"context"
	"fmt"
	"github.com/f1shl3gs/gossiping/cmd/gossiping/internal"
	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
		Use:     "list",
		Short:   "list all jobs",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := internal.ClientFromCmd(cmd)
			var entries []*targetpb.MeshEntry
			err := cli.Get(context.Background(), "/jobs", &entries)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("no jobs")
				return nil
			}

			sort.SliceIsSorted(entries, func(i, j int) bool {
				return entries[i].Name < entries[j].Name
			})

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Name", "Status", "Updated", "Targets", "Labels"})

			for _, ent := range entries {
				table.Append([]string{
					ent.Name,
					targetpb.Status_name[int32(ent.Status)],
					ent.Updated.Local().Format(time.RFC3339),
					strconv.Itoa(len(ent.Targetgroup.Targets)),
					mapToStr(ent.Targetgroup.Labels),
				})
			}

			table.Render()

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

func mapToStr(m map[string]string) string {
	keys := make([]string, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		keys[i] = k + "=" + m[k]
	}

	return strings.Join(keys, ",")
}
