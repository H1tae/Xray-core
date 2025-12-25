package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

type ConnID uint64

var connSeq atomic.Uint64

func NextConnID() ConnID {
	return ConnID(connSeq.Add(1))
}

type ConnInfo struct {
	UUID     string
	ConnID   ConnID
	Started  time.Time
	LastSeen atomic.Int64 // unix seconds
	RxBytes  atomic.Uint64
	TxBytes  atomic.Uint64
}

type Registry struct {
	mu sync.RWMutex
	// uuid -> connID -> *ConnInfo
	byUUID map[string]map[ConnID]*ConnInfo
	// connID -> *ConnInfo
	byConn map[ConnID]*ConnInfo
}

func NewRegistry() *Registry {
	return &Registry{
		byUUID: make(map[string]map[ConnID]*ConnInfo),
		byConn: make(map[ConnID]*ConnInfo),
	}
}

func (r *Registry) Add(uuid string) *ConnInfo {
	cid := NextConnID()
	now := time.Now()

	ci := &ConnInfo{
		UUID:    uuid,
		ConnID:  cid,
		Started: now,
	}
	ci.LastSeen.Store(now.Unix())

	r.mu.Lock()
	defer r.mu.Unlock()

	m, ok := r.byUUID[uuid]
	if !ok {
		m = make(map[ConnID]*ConnInfo)
		r.byUUID[uuid] = m
	}
	m[cid] = ci
	r.byConn[cid] = ci

	return ci
}

func (r *Registry) Remove(connID ConnID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ci, ok := r.byConn[connID]
	if !ok {
		return
	}
	delete(r.byConn, connID)

	if m, ok := r.byUUID[ci.UUID]; ok {
		delete(m, connID)
		if len(m) == 0 {
			delete(r.byUUID, ci.UUID)
		}
	}
}

func (r *Registry) Touch(connID ConnID) {
	r.mu.RLock()
	ci := r.byConn[connID]
	r.mu.RUnlock()
	if ci == nil {
		return
	}
	ci.LastSeen.Store(time.Now().Unix())
}

func (r *Registry) AddRx(connID ConnID, n uint64) {
	r.mu.RLock()
	ci := r.byConn[connID]
	r.mu.RUnlock()
	if ci == nil {
		return
	}
	ci.RxBytes.Add(n)

	ci.LastSeen.Store(time.Now().Unix())
}

func (r *Registry) AddTx(connID ConnID, n uint64) {
	r.mu.RLock()
	ci := r.byConn[connID]
	r.mu.RUnlock()
	if ci == nil {
		return
	}
	ci.TxBytes.Add(n)

	ci.LastSeen.Store(time.Now().Unix())
}

func (r *Registry) ListByUUID(uuid string) []*ConnInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m := r.byUUID[uuid]
	if m == nil {
		return nil
	}
	out := make([]*ConnInfo, 0, len(m))
	for _, ci := range m {
		out = append(out, ci)
	}
	return out
}

func (r *Registry) Get(connID ConnID) *ConnInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byConn[connID]
}
