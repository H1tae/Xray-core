package ratelimit

import (
	"context"
	"sync"

	cctx "github.com/xtls/xray-core/common/ctx"
	"github.com/xtls/xray-core/common/session"
)

// sid -> deviceKey/connID (чтобы один inbound не создавал много ConnID)
var sidIndex = struct {
	mu sync.Mutex
	m  map[uint64]struct {
		deviceKey string
		connID    ConnID
	}
}{
	m: make(map[uint64]struct {
		deviceKey string
		connID    ConnID
	}),
}

// EnsureConnIDFromContext:
// - достаёт uuid + srcIP из session.Inbound
// - достаёт sid из ctx (он создаётся в inbound worker: cctx.ContextWithID)
// - на первый вызов делает DeviceStart(deviceKey, uuid) и ставит cleanup по ctx.Done()
// - дальше возвращает тот же connID для всех outbound внутри этого inbound
func EnsureConnIDFromContext(ctx context.Context) (ConnID, bool) {
	inb := session.InboundFromContext(ctx)
	if inb == nil || inb.User == nil || inb.User.Email == "" {
		return 0, false
	}
	uuid := inb.User.Email

	srcIP := ""
	if inb.Source.IsValid() {
		srcIP = inb.Source.Address.String()
	}
	deviceKey := BuildDeviceKey(uuid, srcIP)

	sid := cctx.IDFromContext(ctx)
	if sid == 0 {
		// Если sid нет, мы не можем гарантировать “один connID на inbound”.
		// Лучше ничего не делать (или потом добавить fallback).
		return 0, false
	}
	sidKey := uint64(sid)

	sidIndex.mu.Lock()
	if e, ok := sidIndex.m[sidKey]; ok {
		sidIndex.mu.Unlock()
		return e.connID, true
	}

	// первый раз для этого inbound
	connID := DeviceStart(deviceKey, uuid)
	sidIndex.m[sidKey] = struct {
		deviceKey string
		connID    ConnID
	}{deviceKey: deviceKey, connID: connID}
	sidIndex.mu.Unlock()

	// cleanup один раз на inbound
	go func() {
		<-ctx.Done()

		DeviceEnd(deviceKey)

		sidIndex.mu.Lock()
		delete(sidIndex.m, sidKey)
		sidIndex.mu.Unlock()
	}()

	return connID, true
}
