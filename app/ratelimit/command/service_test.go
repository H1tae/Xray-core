package command

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xtls/xray-core/app/ratelimit"
	ratelimitpb "github.com/xtls/xray-core/app/ratelimit/api"
)

func TestServiceDeviceMethodsWorkInUUIDMode(t *testing.T) {
	ratelimit.SetKeyMode(ratelimit.KeyModeUUID)

	uuid := fmt.Sprintf("svc-uuid-mode-%d", time.Now().UnixNano())
	deviceKey := ratelimit.BuildDeviceKey(uuid, "198.51.100.24")
	connID := ratelimit.DeviceStart(deviceKey, uuid)
	t.Cleanup(func() {
		ratelimit.DeviceEnd(deviceKey)
		ratelimit.Limits.ClearConnLimit(connID)
		ratelimit.Limits.ClearUserDefault(uuid)
	})

	svc := &Service{}
	ctx := context.Background()

	listResp, err := svc.ListUserDevices(ctx, &ratelimitpb.ListUserDevicesRequest{Uuid: uuid})
	if err != nil {
		t.Fatalf("ListUserDevices returned error: %v", err)
	}
	if len(listResp.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(listResp.Devices))
	}
	if got := listResp.Devices[0]; got.DeviceKey != deviceKey || got.SrcIp != "" || got.ConnId != uint64(connID) {
		t.Fatalf("unexpected device info: %+v", got)
	}

	statsResp, err := svc.GetUserStats(ctx, &ratelimitpb.GetUserStatsRequest{Uuid: uuid})
	if err != nil {
		t.Fatalf("GetUserStats returned error: %v", err)
	}
	if statsResp.DeviceCount != 1 {
		t.Fatalf("expected device count 1, got %d", statsResp.DeviceCount)
	}

	_, err = svc.SetDeviceLimit(ctx, &ratelimitpb.SetDeviceLimitRequest{
		DeviceKey: deviceKey,
		DownBps:   32000,
		UpBps:     16000,
	})
	if err != nil {
		t.Fatalf("SetDeviceLimit returned error: %v", err)
	}

	limit, ok := ratelimit.Limits.GetForConn(uuid, connID)
	if !ok {
		t.Fatal("expected device limit override to be set")
	}
	if limit.Down != 32000 || limit.Up != 16000 {
		t.Fatalf("unexpected limit: %+v", limit)
	}

	_, err = svc.ClearDeviceLimit(ctx, &ratelimitpb.ClearDeviceLimitRequest{DeviceKey: deviceKey})
	if err != nil {
		t.Fatalf("ClearDeviceLimit returned error: %v", err)
	}

	if _, ok := ratelimit.Limits.GetForConn(uuid, connID); ok {
		t.Fatal("expected device limit override to be cleared")
	}
}
