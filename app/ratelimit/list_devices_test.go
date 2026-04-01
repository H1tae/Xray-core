package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

func TestListDevicesByUUIDIncludesUUIDOnlyKey(t *testing.T) {
	SetKeyMode(KeyModeUUID)

	uuid := fmt.Sprintf("uuid-mode-%d", time.Now().UnixNano())
	deviceKey := BuildDeviceKey(uuid, "203.0.113.10")
	if deviceKey != uuid {
		t.Fatalf("expected uuid-only device key, got %q", deviceKey)
	}

	connID := DeviceStart(deviceKey, uuid)
	t.Cleanup(func() {
		DeviceEnd(deviceKey)
	})

	devices := ListDevicesByUUID(uuid)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	got := devices[0]
	if got.UUID != uuid {
		t.Fatalf("expected uuid %q, got %q", uuid, got.UUID)
	}
	if got.DeviceKey != deviceKey {
		t.Fatalf("expected device key %q, got %q", deviceKey, got.DeviceKey)
	}
	if got.SrcIP != "" {
		t.Fatalf("expected empty src ip in uuid mode, got %q", got.SrcIP)
	}
	if got.ConnID != connID {
		t.Fatalf("expected conn id %d, got %d", connID, got.ConnID)
	}
}
