package exporter

import (
	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
)

type Target struct {
	address string

	pinger *ping.Pinger

	// metrics
	recvPackets prometheus.Counter
	sendPackets prometheus.Counter
	rttDuration prometheus.Summary
}

func (target *Target) Describe(descs chan<- *prometheus.Desc) {
	descs <- target.recvPackets.Desc()
	descs <- target.sendPackets.Desc()
	descs <- target.rttDuration.Desc()
}

func (target *Target) Collect(metrics chan<- prometheus.Metric) {
	target.sendPackets.Collect(metrics)
	target.recvPackets.Collect(metrics)
	target.rttDuration.Collect(metrics)
}

func NewTarget(addr string) (*Target, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return nil, err
	}

	constLabels := map[string]string{
		"target": addr,
	}

	recvPackets := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "gossiping",
		Subsystem:   "ping",
		Name:        "recv_packet_total",
		Help:        "The Number of received packets",
		ConstLabels: constLabels,
	})

	sendPackets := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "gossiping",
		Subsystem:   "ping",
		Name:        "send_packet_total",
		Help:        "The number of send packets",
		ConstLabels: constLabels,
	})

	rttDuration := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:   "gossiping",
		Subsystem:   "ping",
		Name:        "rtt_seconds",
		ConstLabels: constLabels,
	})

	pinger.OnSend = func(pkt *ping.Packet) {
		sendPackets.Inc()
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		recvPackets.Inc()
		rttDuration.Observe(pkt.Rtt.Seconds())
	}

	return &Target{
		address:     addr,
		pinger:      pinger,
		recvPackets: recvPackets,
		sendPackets: sendPackets,
		rttDuration: rttDuration,
	}, err
}

func (target *Target) Start() error {
	return target.pinger.Run()
}

func (target *Target) Stop() {
	target.pinger.Stop()
}
