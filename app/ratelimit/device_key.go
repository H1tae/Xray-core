package ratelimit

import (
	"sync"
	"time"
)

type deviceEntry struct {
	id       ConnID
	refCount int
	lastSeen time.Time
	expires  time.Time
	uuid     string

	egressTag string
}

var deviceEntries = struct {
	mu sync.Mutex
	m  map[string]*deviceEntry
}{
	m: make(map[string]*deviceEntry),
}

func DeviceStart(deviceKey string, uuid string) ConnID {
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	now := time.Now()

	if e := deviceEntries.m[deviceKey]; e != nil {
		e.refCount++
		e.lastSeen = now
		e.expires = time.Time{} // сброс удаления
		return e.id
	}

	ci := Global.Add(uuid)
	id := ci.ConnID
	deviceEntries.m[deviceKey] = &deviceEntry{
		id:       id,
		refCount: 1,
		lastSeen: now,
		uuid:     uuid,
	}
	return id
}

func DeviceEnd(deviceKey string) {
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	e := deviceEntries.m[deviceKey]
	if e == nil {
		return
	}
	e.refCount--
	e.lastSeen = time.Now()

	if e.refCount <= 0 {
		// не удаляем сразу — ставим время истечения
		e.refCount = 0
		e.expires = time.Now().Add(GetGrace())
	}
}

func DeviceGetEgress(deviceKey string) (string, bool) {
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	e := deviceEntries.m[deviceKey]
	if e == nil || e.egressTag == "" {
		return "", false
	}
	// entry может быть в grace, но ещё не удалён GC — это ок: привязка действует
	return e.egressTag, true
}

func DeviceSetEgress(deviceKey string, tag string) {
	if tag == "" {
		return
	}
	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	e := deviceEntries.m[deviceKey]
	if e == nil {
		return // если entry нет — привязку пока некуда писать
	}
	e.egressTag = tag
}

func DeviceClearEgressForUUID(uuid string) int {
	if uuid == "" {
		return 0
	}

	deviceEntries.mu.Lock()
	defer deviceEntries.mu.Unlock()

	n := 0
	for _, e := range deviceEntries.m {
		if e != nil && e.uuid == uuid && e.egressTag != "" {
			e.egressTag = ""
			n++
		}
	}
	return n
}
