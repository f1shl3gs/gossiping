package exporter

import "sync"

type state struct {
	mtx sync.RWMutex
}
