package tasks

import (
	"bytes"
	"io"
	"sort"
	"sync"

	"github.com/f1shl3gs/gossiping/tasks/targetpb"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
)

type Store struct {
	mtx       sync.RWMutex
	entries   map[string]*targetpb.MeshEntry
	callbacks map[string]func(me *targetpb.MeshEntry)
}

func NewStore() *Store {
	return &Store{
		entries:   make(map[string]*targetpb.MeshEntry),
		callbacks: make(map[string]func(me *targetpb.MeshEntry)),
	}
}

// todo: may the enable compress for the state
func (s *Store) MarshalBinary() ([]byte, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	buf := bytes.NewBuffer(nil)
	for _, entry := range s.entries {
		_, err := pbutil.WriteDelimited(buf, entry)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (s *Store) Merge(b []byte) error {
	var buf = bytes.NewBuffer(b)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	for {
		var ent targetpb.MeshEntry
		_, err := pbutil.ReadDelimited(buf, &ent)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		if !s.merge(&ent) {
			continue
		}

		for _, cb := range s.callbacks {
			if cb == nil {
				continue
			}

			cb(&ent)
		}
	}
}

func (s *Store) Jobs() []string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	var names []string
	for name := range s.entries {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (s *Store) merge(me *targetpb.MeshEntry) bool {
	prev := s.entries[me.Name]
	if prev == nil {
		s.entries[me.Name] = me
		return true
	}

	if prev.Updated.After(me.Updated) {
		return false
	}

	s.entries[me.Name] = me

	return true
}

func (s *Store) AddCallback(name string, fn func(me *targetpb.MeshEntry)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.callbacks[name] = fn
}
