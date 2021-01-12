package main

import (
	"net/http"
	"os"

	"github.com/f1shl3gs/gossiping/cluster"
	"github.com/f1shl3gs/gossiping/log"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

const (
	defaultHttpAddress = ":9000"
)

const defaultClusterAddr = "0.0.0.0:9094"

/*
clusterBindAddr = kingpin.Flag("cluster.listen-address", "Listen address for cluster. Set to empty string to disable HA mode.").
				Default(defaultClusterAddr).String()
*/

func launch(cmd *cobra.Command, args []string) error {
	logger, err := log.New(os.Stdout)
	if err != nil {
		return err
	}

	peer, err := cluster.Create(
		logger,
		prometheus.DefaultRegisterer,
		defaultClusterAddr,
		"",
		nil,
		true,
		cluster.DefaultPushPullInterval,
		cluster.DefaultGossipInterval,
		cluster.DefaultTcpTimeout,
		cluster.DefaultProbeTimeout,
		cluster.DefaultProbeInterval)
	if err != nil {
		return errors.Wrap(err, "create cluster failed")
	}

	err = peer.Join(cluster.DefaultReconnectInterval, cluster.DefaultReconnectTimeout)
	if err != nil {
		return errors.Wrap(err, "join gossip cluster failed")
	}

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/cluster", func(w http.ResponseWriter, r *http.Request) {

	})

	return nil
}
