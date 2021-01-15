package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	host string
}

func ClientFromCmd(cmd *cobra.Command) *Client {
	val := cmd.Flags().Lookup("host")
	if val == nil {
		panic("host flag is not defined")
	}

	return &Client{
		host: val.Value.String(),
	}
}

func (cli *Client) Get(ctx context.Context, url string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cli.host+url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if dst == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(dst)
}

func (cli *Client) Post(ctx context.Context, url string, payload interface{}) error {
	if payload == nil {
		return errors.New("payload cannot be empty")
	}

	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		return err
	}

	return cli.PostWithReader(ctx, url, buf)
}

func (cli *Client) PostWithReader(ctx context.Context, url string, r io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cli.host+url, r)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(resp.Body)
		return errors.Errorf("unexpected status code %d, resp: %s", resp.StatusCode, data)
	}

	return nil
}
