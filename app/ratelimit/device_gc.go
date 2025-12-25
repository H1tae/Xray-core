package ratelimit

import (
	"time"
	
)

func init() {
	go deviceGC()
}

func deviceGC() {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for range t.C {
		now := time.Now()

		var toDelete []ConnID

		deviceEntries.mu.Lock()
		for key, e := range deviceEntries.m {
			if e == nil {
				delete(deviceEntries.m, key)
				continue
			}
			if e.refCount == 0 && !e.expires.IsZero() && now.After(e.expires) {
				delete(deviceEntries.m, key)
				toDelete = append(toDelete, e.id)
			}
		}
		deviceEntries.mu.Unlock()

		// чистим вне lock
		for _, id := range toDelete {
			buckets.Remove(id)
			Global.Remove(id)
		}
	}
}
