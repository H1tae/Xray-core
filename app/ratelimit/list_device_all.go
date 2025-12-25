package ratelimit

import (
	"strings"
	"time"
)

func ListDevicesAll() []DeviceSnapshot {
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	out := make([]DeviceSnapshot, 0, len(deviceEntries.m))

	for key, e := range deviceEntries.m {
		if e == nil {
			continue
		}

		// key = uuid|srcIP
		uuid := key
		srcIP := ""
		if i := strings.IndexByte(key, '|'); i >= 0 {
			uuid = key[:i]
			srcIP = key[i+1:]
		}

		var started time.Time
		var lastSeen time.Time
		var rx, tx uint64

		if ci := Global.Get(e.id); ci != nil {
			started = ci.Started
			lastSeen = time.Unix(ci.LastSeen.Load(), 0)
			rx = ci.RxBytes.Load()
			tx = ci.TxBytes.Load()
		} else {
			started = e.lastSeen
			lastSeen = e.lastSeen
		}

		out = append(out, DeviceSnapshot{
			UUID:      uuid,
			SrcIP:     srcIP,
			DeviceKey: key,
			ConnID:    e.id,
			RefCount:  uint32(e.refCount),
			StartedAt: started,
			LastSeen:  lastSeen,
			RxBytes:   rx,
			TxBytes:   tx,
		})
	}

	return out
}
