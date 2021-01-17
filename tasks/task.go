package tasks

import (
	"time"

	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Task struct {
	address string

	pinger  *ping.Pinger
	stopped bool

	// metrics
	recvPackets prometheus.Counter
	sendPackets prometheus.Counter
	rttDuration prometheus.Summary
	pingError   prometheus.Gauge
}

func (task *Task) Describe(descs chan<- *prometheus.Desc) {
	descs <- task.recvPackets.Desc()
	descs <- task.sendPackets.Desc()
	descs <- task.rttDuration.Desc()
	descs <- task.pingError.Desc()
}

func (task *Task) Collect(metrics chan<- prometheus.Metric) {
	task.sendPackets.Collect(metrics)
	task.recvPackets.Collect(metrics)
	task.rttDuration.Collect(metrics)
	task.pingError.Collect(metrics)
}

func newTask(addr string, lbs map[string]string) (*Task, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return nil, err
	}
	pinger.SetPrivileged(true)

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

	pingError := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "gossiping",
		Subsystem:   "ping",
		Name:        "error",
		ConstLabels: constLabels,
	})

	pinger.OnSend = func(pkt *ping.Packet) {
		pingError.Set(0)
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
		pingError:   pingError,
	}, err
}

func (task *Task) Start(logger *zap.Logger) {
	defer func() {
		err := recover()
		if err != nil {
			logger.Error("task panicked",
				zap.Stack("task"))
		}
	}()

	task.pingError.Set(1)
	for {
		err := task.pinger.Run()
		if task.stopped {
			return
		}

		task.pingError.Set(1)
		if err != nil {
			logger.Warn("ping error",
				zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (task *Task) Stop() {
	task.stopped = true
	task.pinger.Stop()
}
