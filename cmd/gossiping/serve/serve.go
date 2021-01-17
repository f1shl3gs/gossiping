package serve

import (
	"io/ioutil"

	"github.com/f1shl3gs/gossiping/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func Serve() *cobra.Command {
	var (
		confPath string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start gossiping daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := ioutil.ReadFile(confPath)
			if err != nil {
				return err
			}

			conf := config.Config{}
			err = yaml.UnmarshalStrict(data, &conf)
			if err != nil {
				return err
			}

			return launch(conf)
		},
	}

	cmd.Flags().StringVarP(&confPath, "config", "c",
		"/etc/gossiping/gossiping.yml", "config file path")

	return cmd
}
