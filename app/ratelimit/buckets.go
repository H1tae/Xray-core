package ratelimit

import "sync"

type Buckets struct {
	mu sync.Mutex
	up   map[ConnID]*TokenBucket
	down map[ConnID]*TokenBucket
}

func NewBuckets() *Buckets {
	return &Buckets{
		up:   make(map[ConnID]*TokenBucket),
		down: make(map[ConnID]*TokenBucket),
	}
}

var buckets = NewBuckets()

func bpsToBytesPerSec(bps uint64) float64 {
	// bps = bits/sec
	return float64(bps) / 8.0
}

// GetOrCreate возвращает up/down bucket для conn_id и подстраивает rate под текущие лимиты.
func (b *Buckets) GetOrCreate(conn ConnID, upBps, downBps uint64) (*TokenBucket, *TokenBucket) {
	b.mu.Lock()
	defer b.mu.Unlock()

	up := b.up[conn]
	if up == nil {
		up = NewTokenBucket(bpsToBytesPerSec(upBps))
		b.up[conn] = up
	} else {
		up.SetRate(bpsToBytesPerSec(upBps))
	}

	down := b.down[conn]
	if down == nil {
		down = NewTokenBucket(bpsToBytesPerSec(downBps))
		b.down[conn] = down
	} else {
		down.SetRate(bpsToBytesPerSec(downBps))
	}

	return up, down
}

func (b *Buckets) Remove(conn ConnID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.up, conn)
	delete(b.down, conn)
}
