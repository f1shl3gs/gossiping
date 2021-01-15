package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/f1shl3gs/gossiping/cluster"
	"github.com/f1shl3gs/gossiping/log"
	"github.com/f1shl3gs/gossiping/pkg/signals"
	"github.com/f1shl3gs/gossiping/tasks"
	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"github.com/julienschmidt/httprouter"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	defaultHttpAddress = ":9000"
	defaultClusterAddr = "0.0.0.0:9094"
)

func launch(cmd *cobra.Command, args []string) error {
	logger, err := log.New(os.Stdout)
	if err != nil {
		return err
	}

	defer logger.Sync()

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

	defer func() {
		err = peer.Leave(3 * time.Second)
		if err != nil {
			logger.Error("peer leave failed",
				zap.Error(err))
		}
	}()

	store := tasks.NewStore()
	ch := peer.AddState("tg", store, prometheus.DefaultRegisterer)
	broadcast := func(me *targetpb.MeshEntry) error {
		buf := bytes.NewBuffer(nil)
		_, err := pbutil.WriteDelimited(buf, me)
		if err != nil {
			return err
		}

		ch.Broadcast(buf.Bytes())

		return err
	}

	// collector
	collector := tasks.New(logger)
	prometheus.MustRegister(collector)

	store.AddCallback("collector", collector.Coordinate)

	store.AddCallback("state", func(me *targetpb.MeshEntry) {
		data, err := json.MarshalIndent(me, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(data))
	})

	router := httprouter.New()
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())
	router.Handler(http.MethodGet, "/debug/pprof/*dummy", http.DefaultServeMux)
	router.HandlerFunc(http.MethodGet, "/cluster", func(w http.ResponseWriter, r *http.Request) {
		ps := peer.Peers()
		err := json.NewEncoder(w).Encode(&ps)
		if err != nil {
			logger.Warn("encode peers failed",
				zap.String("remote", r.RemoteAddr),
				zap.Error(err))
		}
	})

	// list jobs
	router.HandlerFunc(http.MethodGet, "/jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs := store.Jobs()
		err = json.NewEncoder(w).Encode(&jobs)
		if err != nil {
			logger.Warn("write jobs to client failed",
				zap.String("remote", r.RemoteAddr),
				zap.Error(err))
		}
	})

	// add jobs
	router.HandlerFunc(http.MethodPost, "/jobs/:name", func(w http.ResponseWriter, r *http.Request) {
		params := httprouter.ParamsFromContext(r.Context())
		name := params.ByName("name")

		var tg targetgroup.Group
		err = json.NewDecoder(r.Body).Decode(&tg)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		err = broadcast(&targetpb.MeshEntry{
			Name:        name,
			Status:      targetpb.Status_Active,
			Updated:     time.Now(),
			Targetgroup: targetpb.FromProm(&tg),
		})
		if err != nil {
			logger.Warn("add job failed",
				zap.Error(err))
		}
	})

	// delete job by name
	router.HandlerFunc(http.MethodDelete, "/jobs/:name", func(w http.ResponseWriter, r *http.Request) {
		params := httprouter.ParamsFromContext(r.Context())
		name := params.ByName("name")

		err = broadcast(&targetpb.MeshEntry{
			Name:        name,
			Status:      targetpb.Status_Inactive,
			Updated:     time.Now(),
			Targetgroup: nil,
		})
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	})

	ctx := signals.WithStandardSignals(context.Background())
	group, ctx := errgroup.WithContext(ctx)

	// http server
	group.Go(func() error {
		server := http.Server{
			Addr:    defaultHttpAddress,
			Handler: router,
		}

		errCh := make(chan error)
		go func() {
			errCh <- server.ListenAndServe()
		}()

		select {
		case <-ctx.Done():
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := server.Shutdown(ctx)
			if err != nil {
				logger.Warn("shutdown http server failed",
					zap.Error(err))
			}

			return err
		case err := <-errCh:
			logger.Warn(" start http server failed",
				zap.Error(err))
			return err
		}
	})

	return group.Wait()
}
