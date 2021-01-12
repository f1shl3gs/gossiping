package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
)

func ClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "CRUD cluster information",
	}

	cmd.PersistentFlags().String("host", "localhost:9000", "address of gossiping daemon to interact")

	cmd.AddCommand(joinCmd())

	return cmd
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list all members with their metadata",
		RunE: func(cmd *cobra.Command, args []string) error {

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
