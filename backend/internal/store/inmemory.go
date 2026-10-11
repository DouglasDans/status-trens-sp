// Package store keeps each line's status.Memory between polls.
package store

import (
	"context"
	"maps"
	"sync"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// InMemory is a store that lives as long as the process: local runs and
// tests. A restart forgets everything, so every line starts a new baseline.
type InMemory struct {
	mu    sync.Mutex
	lines map[int]status.Memory
}

func NewInMemory() *InMemory {
	return &InMemory{lines: map[int]status.Memory{}}
}

// Load returns the memory of every line seen so far, by line code.
func (s *InMemory) Load(context.Context) (map[int]status.Memory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.lines), nil
}

func (s *InMemory) Save(_ context.Context, code int, m status.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines[code] = m
	return nil
}
