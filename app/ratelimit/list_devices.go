package ratelimit

import (
	"strings"
	"time"
)

type DeviceSnapshot struct {
	UUID      string
	SrcIP     string
	DeviceKey string
	ConnID    ConnID
	RefCount  uint32
	StartedAt time.Time
	LastSeen  time.Time
	RxBytes   uint64
	TxBytes   uint64
}

func ListDevicesByUUID(uuid string) []DeviceSnapshot {
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	out := make([]DeviceSnapshot, 0, 8)

	prefix := uuid + "|"

	for key, e := range deviceEntries.m {
		if e == nil {
			continue
		}
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		srcIP := strings.TrimPrefix(key, prefix)

		// Достаём counters из Global по connID (если есть)
		var rx, tx uint64
		var startedAt time.Time
		var lastSeen time.Time
		if ci := Global.Get(e.id); ci != nil { // если у тебя нет Global.Get — скажи, я подстрою под твою реализацию
			rx = uint64(ci.RxBytes.Load())
			tx = uint64(ci.TxBytes.Load())
			startedAt = ci.Started
			lastSeen = time.Unix(ci.LastSeen.Load(), 0)
		} else {
			// fallback на поля entry
			startedAt = e.lastSeen
			lastSeen = e.lastSeen
		}

		out = append(out, DeviceSnapshot{
			UUID:      uuid,
			SrcIP:     srcIP,
			DeviceKey: key,
			ConnID:    e.id,
			RefCount:  uint32(e.refCount),
			StartedAt: startedAt,
			LastSeen:  lastSeen,
			RxBytes:   rx,
			TxBytes:   tx,
		})
	}

	return out
}
