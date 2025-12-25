package ratelimit

import "sync"

type RateBps struct {
	Down uint64
	Up   uint64
}

type LimitStore struct {
	mu sync.RWMutex
	// uuid -> default per-conn limit
	defaultPerConn map[string]RateBps
	// conn_id overrides
	overrides map[ConnID]RateBps
}

func NewLimitStore() *LimitStore {
	return &LimitStore{
		defaultPerConn: make(map[string]RateBps),
		overrides:      make(map[ConnID]RateBps),
	}
}

var Limits = NewLimitStore()

func (s *LimitStore) SetUserDefault(uuid string, down, up uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultPerConn[uuid] = RateBps{Down: down, Up: up}
}

func (s *LimitStore) SetConnLimit(conn ConnID, down, up uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overrides[conn] = RateBps{Down: down, Up: up}
}

func (s *LimitStore) ClearConnLimit(conn ConnID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.overrides, conn)
}

func (s *LimitStore) GetForConn(uuid string, conn ConnID) (RateBps, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.overrides[conn]; ok {
		return v, true
	}
	v, ok := s.defaultPerConn[uuid]
	return v, ok
}
