package tasks

import (
	"sort"
	"sync"

	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Collector struct {
	logger *zap.Logger

	mtx   sync.RWMutex
	tasks map[string]map[uint64]*Task

	// state descs

}

func New(logger *zap.Logger) *Collector {
	c := &Collector{
		logger: logger,
		tasks:  make(map[string]map[uint64]*Task),
	}

	return c
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	descs <- prometheus.NewDesc("dummy", "", nil, nil)
}

func (c *Collector) Collect(metrics chan<- prometheus.Metric) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	for _, group := range c.tasks {
		for _, task := range group {
			task.Collect(metrics)
		}
	}
}

func (c *Collector) Coordinate(me *targetpb.MeshEntry) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	taskGroup := c.tasks[me.Name]
	if taskGroup == nil {
		taskGroup = make(map[uint64]*Task)
		c.tasks[me.Name] = taskGroup
	}

	// add task
	idCache := make([]uint64, 0, len(me.Targetgroup.Targets))
	for _, addr := range me.Targetgroup.Targets {
		taskID := TaskID(addr, me.Targetgroup.Labels)
		idCache = append(idCache, taskID)
		_, ok := taskGroup[taskID]
		if ok {
			continue
		}

		task, err := newTask(addr, me.Targetgroup.Labels)
		if err != nil {
			c.logger.Warn("create new task failed",
				zap.String("job", me.Name),
				zap.String("addr", addr),
				zap.Error(err))
			continue
		}

		if err := task.Start(); err != nil {
			c.logger.Warn("start task failed",
				zap.String("job", me.Name),
				zap.String("addr", addr),
				zap.Error(err))
		}

		taskGroup[taskID] = task
		c.logger.Info("add target",
			zap.String("job", me.Name),
			zap.String("addr", addr))
	}

	// remote none exist
	for taskID, task := range taskGroup {
		found := false
		for _, id := range idCache {
			if id == taskID {
				found = true
				break
			}
		}

		if found {
			continue
		}

		task.Stop()
		delete(taskGroup, taskID)
		c.logger.Info("delete target",
			zap.String("job", me.Name),
			zap.String("addr", task.address))
	}
}

// SeparatorByte is a byte that cannot occur in valid UTF-8 sequences and is
// used to separate label names, label values, and other strings from each other
// when calculating their combined hash value (aka signature aka fingerprint).
const SeparatorByte byte = 255

var (
	// cache the signature of an empty label set.
	emptyLabelSignature = hashNew()
)

func TaskID(addr string, labels map[string]string) uint64 {
	if len(labels) == 0 {
		return emptyLabelSignature
	}

	labelNames := make([]string, 0, len(labels))
	for labelName := range labels {
		labelNames = append(labelNames, labelName)
	}
	sort.Strings(labelNames)

	sum := hashNew()
	for _, labelName := range labelNames {
		sum = hashAdd(sum, labelName)
		sum = hashAddByte(sum, SeparatorByte)
		sum = hashAdd(sum, labels[labelName])
		sum = hashAddByte(sum, SeparatorByte)
	}

	sum = hashAdd(sum, "__addr__")
	sum = hashAddByte(sum, SeparatorByte)
	sum = hashAdd(sum, addr)
	sum = hashAddByte(sum, SeparatorByte)

	return sum
}
