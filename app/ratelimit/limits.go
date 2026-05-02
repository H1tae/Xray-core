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

func (s *LimitStore) SetUserDefaults(limits map[string]RateBps) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := 0
	for uuid, limit := range limits {
		if uuid == "" {
			continue
		}
		s.defaultPerConn[uuid] = limit
		updated++
	}
	return updated
}

func (s *LimitStore) ClearUserDefault(uuid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.defaultPerConn, uuid)
}

func (s *LimitStore) ClearUserDefaults(uuids []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleared := 0
	for _, uuid := range uuids {
		if uuid == "" {
			continue
		}
		if _, ok := s.defaultPerConn[uuid]; ok {
			delete(s.defaultPerConn, uuid)
			cleared++
		}
	}
	return cleared
}

func (s *LimitStore) ClearAll() (defaults int, overrides int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaults = len(s.defaultPerConn)
	overrides = len(s.overrides)
	s.defaultPerConn = make(map[string]RateBps)
	s.overrides = make(map[ConnID]RateBps)
	return defaults, overrides
}

func ClearUserOverride(uuid string) int {
	return ClearUserOverrides([]string{uuid})
}

// ClearUserOverrides удаляет ВСЕ overrides (per-conn) для указанных uuid.
// Возвращает количество очищенных записей.
func ClearUserOverrides(uuids []string) int {
	connIDs := make(map[ConnID]struct{})
	for _, uuid := range uuids {
		if uuid == "" {
			continue
		}
		for _, ci := range Global.ListByUUID(uuid) {
			connIDs[ci.ConnID] = struct{}{}
		}
	}

	if len(connIDs) == 0 {
		return 0
	}

	s := Limits // глобальный store
	s.mu.Lock()
	defer s.mu.Unlock()

	cleared := 0
	for connID := range connIDs {
		if _, ok := s.overrides[connID]; ok {
			delete(s.overrides, connID)
			cleared++
		}
	}

	return cleared
}

func ClearUserRateLimits(uuids []string) (defaults int, overrides int) {
	defaults = Limits.ClearUserDefaults(uuids)
	overrides = ClearUserOverrides(uuids)
	return defaults, overrides
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
