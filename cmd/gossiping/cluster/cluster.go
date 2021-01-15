package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/f1shl3gs/gossiping/cmd/gossiping/internal"
	"github.com/hashicorp/memberlist"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "CRUD cluster information",
	}

	cmd.PersistentFlags().String("host", "http://localhost:9000", "address of gossiping daemon to interact")

	cmd.AddCommand(joinCmd())
	cmd.AddCommand(listCmd())

	return cmd
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list all members with their metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := internal.ClientFromCmd(cmd)
			nodes := make([]*memberlist.Node, 0)
			err := cli.Get(context.Background(), "/cluster", &nodes)
			if err != nil {
				return err
			}

			w := tablewriter.NewWriter(os.Stdout)
			w.SetAutoFormatHeaders(false)
			w.SetHeader([]string{"Name", "Addr", "Port", "Meta", "PMin", "PMax", "PCur", "DMin", "DMax", "DCur"})
			for i := 0; i < len(nodes); i++ {
				w.Append([]string{
					nodes[i].Name,
					nodes[i].Addr.String(),
					strconv.Itoa(int(nodes[i].Port)),
					string(nodes[i].Meta),
					strconv.Itoa(int(nodes[i].PMin)),
					strconv.Itoa(int(nodes[i].PMax)),
					strconv.Itoa(int(nodes[i].PCur)),
					strconv.Itoa(int(nodes[i].DMin)),
					strconv.Itoa(int(nodes[i].DMax)),
					strconv.Itoa(int(nodes[i].DCur)),
				})
			}

			w.Render()

			return nil
		},
	}

	return cmd
}

func joinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "join to the gossip cluster",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, err := cmd.PersistentFlags().GetString("host")
			if err != nil {
				return err
			}

			buf := bytes.NewBuffer(nil)
			err = json.NewEncoder(buf).Encode(&args)
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/cluster", host)
			resp, err := http.Post(url, "application/json", buf)
			if err != nil {
				return err
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return errors.Errorf("unexpected status code, %d", resp.StatusCode)
			}

			return nil
		},
	}

	return cmd
}
