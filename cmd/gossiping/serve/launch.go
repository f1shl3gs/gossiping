package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/f1shl3gs/gossiping/cluster"
	"github.com/f1shl3gs/gossiping/config"
	"github.com/f1shl3gs/gossiping/log"
	"github.com/f1shl3gs/gossiping/pkg/signals"
	"github.com/f1shl3gs/gossiping/state"
	"github.com/f1shl3gs/gossiping/tasks"
	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"github.com/julienschmidt/httprouter"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

const (
	defaultHttpAddress = ":9000"
	defaultClusterAddr = "0.0.0.0:9094"
)

func launch(conf config.Config) error {
	logger, err := log.New(os.Stdout)
	if err != nil {
		return err
	}

	defer logger.Sync()

	peer, err := cluster.Create(
		logger,
		prometheus.DefaultRegisterer,
		defaultClusterAddr,
		conf.Cluster.AdvertiseAddr,
		conf.Cluster.Peers,
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
		logger.Warn("errors occurred when join cluster",
			zap.Error(err))
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
	collector := tasks.New(logger, conf.Global.ExternalLabels)
	prometheus.MustRegister(collector)

	if !conf.Tasks.DryRun {
		store.AddCallback("collector", collector.Coordinate)
	} else {
		logger.Info("dry run is enabled for tasks")
	}

	// states
	if conf.Tasks.States != "" {
		logger.Info("task states is enabled",
			zap.String("dir", conf.Tasks.States))

		gen := state.New(conf.Tasks.States, logger)
		store.AddCallback("state", gen.OnUpdate)
	}

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
		err := store.Snapshot(w)
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

	if conf.Prometheus.Output != "" {
		logger.Info("prometheus sd is configured",
			zap.String("filepath", conf.Prometheus.Output))
	}

	group.Go(func() error {
		if err := updateGossipingJob(peer, broadcast); err != nil {
			logger.Warn("initial gossiping job failed",
				zap.Error(err))
		}

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			case <-peer.Changed():
			}

			err = updateGossipingJob(peer, broadcast)
			if err != nil {
				logger.Warn("update gossiping job failed",
					zap.Error(err))
			} else {
				logger.Info("update gossiping job success")
			}

			if conf.Prometheus.Output != "" {
				logger.Info("prometheus sd is configured",
					zap.String("filepath", conf.Prometheus.Output))

				err := generatePromConfig(peer, conf.Prometheus.Output)
				if err != nil {
					logger.Warn("generate prometheus sd file failed",
						zap.Error(err))
				} else {
					logger.Info("regenerate prometheus sd file success")
				}
			}
		}
	})

	return group.Wait()
}

func updateGossipingJob(peer *cluster.Peer, broadcast func(me *targetpb.MeshEntry) error) error {
	if peer.Position() != 0 {
		return nil
	}

	me := &targetpb.MeshEntry{
		Name:        "__gossiping",
		Status:      targetpb.Status_Active,
		Updated:     time.Now(),
		Targetgroup: &targetpb.Targetgroup{},
	}

	for _, p := range peer.Peers() {
		me.Targetgroup.Targets = append(me.Targetgroup.Targets, p.Addr.String())
	}

	return broadcast(me)
}

func generatePromConfig(peer *cluster.Peer, output string) error {
	f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString("# This file is generated by Gossiping, DO NOT EDIT IT\n\n")
	if err != nil {
		return err
	}

	promConf := struct {
		Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
		Targets []string          `json:"targets,omitempty" yaml:"targets,omitempty"`
	}{
		Labels:  map[string]string{},
		Targets: make([]string, 0),
	}

	for _, p := range peer.Peers() {
		promConf.Targets = append(promConf.Targets, p.Addr.String()+defaultHttpAddress)
	}

	return yaml.NewEncoder(f).Encode(&promConf)
}

/*
targets:
- "10.111.222.167",
- "10.111.87.249",
- "10.111.227.136",
- "10.111.90.215",
- "10.111.26.17",
labels: {}
*/
