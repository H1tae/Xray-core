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

func (s *LimitStore) ClearUserDefault(uuid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.defaultPerConn, uuid)
}

func (s *LimitStore) ClearUserOverride(uuid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.defaultPerConn, uuid)
}

// ClearUserOverride удаляет ВСЕ overrides (per-conn) для указанного uuid.
// Возвращает количество очищенных записей.
func ClearUserOverride(uuid string) int {
	if uuid == "" {
		return 0
	}

	// Берём все connID, которые сейчас принадлежат этому uuid
	conns := Global.ListByUUID(uuid)
	if len(conns) == 0 {
		return 0
	}

	s := Limits // глобальный store
	s.mu.Lock()
	defer s.mu.Unlock()

	cleared := 0
	for _, ci := range conns {
		if _, ok := s.overrides[ci.ConnID]; ok {
			delete(s.overrides, ci.ConnID)
			cleared++
		}
	}

	return cleared
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
