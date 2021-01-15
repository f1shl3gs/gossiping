package tasks

import (
	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
)

type Task struct {
	address string

	pinger *ping.Pinger

	// metrics
	recvPackets prometheus.Counter
	sendPackets prometheus.Counter
	rttDuration prometheus.Summary
}

func (task *Task) Describe(descs chan<- *prometheus.Desc) {
	descs <- task.recvPackets.Desc()
	descs <- task.sendPackets.Desc()
	descs <- task.rttDuration.Desc()
}

func (task *Task) Collect(metrics chan<- prometheus.Metric) {
	task.sendPackets.Collect(metrics)
	task.recvPackets.Collect(metrics)
	task.rttDuration.Collect(metrics)
}

func newTask(addr string, lbs map[string]string) (*Task, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return nil, err
	}

	constLabels := make(map[string]string, len(lbs)+1)
	for k, v := range lbs {
		constLabels[k] = v
	}
	constLabels["target"] = addr

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

	return &Task{
		address:     addr,
		pinger:      pinger,
		recvPackets: recvPackets,
		sendPackets: sendPackets,
		rttDuration: rttDuration,
	}, err
}

func (task *Task) Start() error {
	return task.pinger.Run()
}

func (task *Task) Stop() {
	task.pinger.Stop()
}
