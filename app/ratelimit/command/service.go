package command

import (
	"context"
	"strings"
	"time"

	"github.com/xtls/xray-core/app/commander"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"google.golang.org/grpc"

	"github.com/xtls/xray-core/app/ratelimit"
	ratelimitpb "github.com/xtls/xray-core/app/ratelimit/api"
)

// Service одновременно:
// 1) реализует commander.Service (чтобы Commander мог Register() вызвать)
// 2) реализует gRPC сервер ratelimitpb.RateLimitServiceServer (RPC методы)
type Service struct {
	ratelimitpb.UnimplementedRateLimitServiceServer
}

// Register — это то, что вызывает Commander при старте
func (s *Service) Register(gs *grpc.Server) {
	errors.LogInfo(context.Background(), "ratelimit/command: Register() called")
	ratelimitpb.RegisterRateLimitServiceServer(gs, s)
}

// ---- RPC методы ----

func (s *Service) ClearUserEgressCache(
	ctx context.Context,
	req *ratelimitpb.ClearUserEgressCacheRequest,
) (*ratelimitpb.ClearUserEgressCacheResponse, error) {

	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}

	cleared := ratelimit.DeviceClearEgressForUUID(req.Uuid)
	return &ratelimitpb.ClearUserEgressCacheResponse{
		Cleared: uint32(cleared),
	}, nil
}

func (s *Service) SetGrace(ctx context.Context, req *ratelimitpb.SetGraceRequest) (*ratelimitpb.SetGraceResponse, error) {
	ratelimit.SetGrace(time.Duration(req.Seconds) * time.Second)
	return &ratelimitpb.SetGraceResponse{}, nil
}

func (s *Service) GetGrace(ctx context.Context, req *ratelimitpb.GetGraceRequest) (*ratelimitpb.GetGraceResponse, error) {
	sec := uint32(ratelimit.GetGrace().Seconds())
	return &ratelimitpb.GetGraceResponse{Seconds: sec}, nil
}

func (s *Service) SetKeyMode(ctx context.Context, req *ratelimitpb.SetKeyModeRequest) (*ratelimitpb.SetKeyModeResponse, error) {
	switch req.Mode {
	case ratelimitpb.SetKeyModeRequest_UUID:
		ratelimit.SetKeyMode(ratelimit.KeyModeUUID)
	default:
		ratelimit.SetKeyMode(ratelimit.KeyModeDevice)
	}
	return &ratelimitpb.SetKeyModeResponse{}, nil
}

func (s *Service) GetKeyMode(ctx context.Context, req *ratelimitpb.GetKeyModeRequest) (*ratelimitpb.GetKeyModeResponse, error) {
	m := ratelimitpb.SetKeyModeRequest_DEVICE
	if ratelimit.GetKeyMode() == ratelimit.KeyModeUUID {
		m = ratelimitpb.SetKeyModeRequest_UUID
	}
	return &ratelimitpb.GetKeyModeResponse{Mode: m}, nil
}

func (s *Service) GetUserStats(ctx context.Context, req *ratelimitpb.GetUserStatsRequest) (*ratelimitpb.GetUserStatsResponse, error) {
	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}

	devs := ratelimit.ListDevicesByUUID(req.Uuid)
	if len(devs) == 0 {
		return &ratelimitpb.GetUserStatsResponse{Uuid: req.Uuid, DeviceCount: 0}, nil
	}

	var rx, tx uint64
	minStarted := devs[0].StartedAt.Unix()
	maxLastSeen := devs[0].LastSeen.Unix()

	for _, d := range devs {
		rx += d.RxBytes
		tx += d.TxBytes

		if s := d.StartedAt.Unix(); s < minStarted {
			minStarted = s
		}
		if l := d.LastSeen.Unix(); l > maxLastSeen {
			maxLastSeen = l
		}
	}

	return &ratelimitpb.GetUserStatsResponse{
		Uuid:             req.Uuid,
		DeviceCount:      uint32(len(devs)),
		RxBytesTotal:     rx,
		TxBytesTotal:     tx,
		StartedAtUnixMin: uint64(minStarted),
		LastSeenUnixMax:  uint64(maxLastSeen),
	}, nil
}

func (s *Service) GetActiveDevicesSnapshot(ctx context.Context, req *ratelimitpb.GetActiveDevicesSnapshotRequest) (*ratelimitpb.GetActiveDevicesSnapshotResponse, error) {
	devs := ratelimit.ListDevicesAll()

	resp := &ratelimitpb.GetActiveDevicesSnapshotResponse{
		Devices: make([]*ratelimitpb.DeviceInfo, 0, len(devs)),
	}

	for _, d := range devs {
		// Очищаем статистику атомарно в registry (где живут ConnInfo)
		var rx, tx uint64
		if ci := ratelimit.Global.Get(d.ConnID); ci != nil {
			rx = ci.RxBytes.Swap(0)
			tx = ci.TxBytes.Swap(0)
		} else {
			// если ConnInfo уже удалён, отдаём то, что было в снапшоте
			rx = d.RxBytes
			tx = d.TxBytes
		}

		resp.Devices = append(resp.Devices, &ratelimitpb.DeviceInfo{
			Uuid:          d.UUID,
			SrcIp:         d.SrcIP,
			DeviceKey:     d.DeviceKey,
			ConnId:        uint64(d.ConnID),
			RefCount:      d.RefCount,
			StartedAtUnix: uint64(d.StartedAt.Unix()),
			LastSeenUnix:  uint64(d.LastSeen.Unix()),
			RxBytes:       rx,
			TxBytes:       tx,
		})
	}

	return resp, nil
}

func (s *Service) PeekActiveDevicesSnapshot(ctx context.Context, req *ratelimitpb.GetActiveDevicesSnapshotRequest) (*ratelimitpb.GetActiveDevicesSnapshotResponse, error) {
	devs := ratelimit.ListDevicesAll()

	resp := &ratelimitpb.GetActiveDevicesSnapshotResponse{
		Devices: make([]*ratelimitpb.DeviceInfo, 0, len(devs)),
	}

	for _, d := range devs {
		// НЕ очищаем: читаем текущие значения через Load()
		var rx, tx uint64
		if ci := ratelimit.Global.Get(d.ConnID); ci != nil {
			rx = ci.RxBytes.Load()
			tx = ci.TxBytes.Load()
		} else {
			// если ConnInfo уже удалён — берём что было в снапшоте
			rx = d.RxBytes
			tx = d.TxBytes
		}

		resp.Devices = append(resp.Devices, &ratelimitpb.DeviceInfo{
			Uuid:          d.UUID,
			SrcIp:         d.SrcIP,
			DeviceKey:     d.DeviceKey,
			ConnId:        uint64(d.ConnID),
			RefCount:      d.RefCount,
			StartedAtUnix: uint64(d.StartedAt.Unix()),
			LastSeenUnix:  uint64(d.LastSeen.Unix()),
			RxBytes:       rx,
			TxBytes:       tx,
		})
	}

	return resp, nil
}

func (s *Service) SetUserTotalLimit(
	ctx context.Context,
	req *ratelimitpb.SetUserTotalLimitRequest,
) (*ratelimitpb.SetUserTotalLimitResponse, error) {

	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}

	devices := ratelimit.ListDevicesByUUID(req.Uuid)
	n := len(devices)

	if n == 0 {
		return &ratelimitpb.SetUserTotalLimitResponse{
			DeviceCount: 0,
		}, nil
	}

	down := req.DownBps / uint64(n)
	up := req.UpBps / uint64(n)

	for _, d := range devices {
		ratelimit.Limits.SetConnLimit(d.ConnID, down, up)
	}

	return &ratelimitpb.SetUserTotalLimitResponse{
		DeviceCount:      uint32(n),
		PerDeviceDownBps: down,
		PerDeviceUpBps:   up,
	}, nil
}

func (s *Service) ListUserConnections(ctx context.Context, req *ratelimitpb.ListUserConnectionsRequest) (*ratelimitpb.ListUserConnectionsResponse, error) {
	conns := ratelimit.Global.ListByUUID(req.Uuid)

	resp := &ratelimitpb.ListUserConnectionsResponse{
		Connections: make([]*ratelimitpb.ConnectionInfo, 0, len(conns)),
	}
	for _, c := range conns {
		resp.Connections = append(resp.Connections, &ratelimitpb.ConnectionInfo{
			ConnId:        uint64(c.ConnID),
			StartedAtUnix: uint64(c.Started.Unix()),
			LastSeenUnix:  uint64(c.LastSeen.Load()),
			RxBytes:       c.RxBytes.Load(),
			TxBytes:       c.TxBytes.Load(),
		})
	}
	return resp, nil
}

func (s *Service) ClearUserDefaultPerConnLimit(
	ctx context.Context,
	req *ratelimitpb.ClearUserDefaultPerConnLimitRequest,
) (*ratelimitpb.ClearUserDefaultPerConnLimitResponse, error) {

	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}

	ratelimit.Limits.ClearUserDefault(req.Uuid)

	return &ratelimitpb.ClearUserDefaultPerConnLimitResponse{}, nil
}

func (s *Service) ClearUserConnOverrideLimits(
	ctx context.Context,
	req *ratelimitpb.ClearUserConnOverrideLimitsRequest,
) (*ratelimitpb.ClearUserConnOverrideLimitsResponse, error) {

	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}

	cleared := ratelimit.ClearUserOverride(req.Uuid)

	return &ratelimitpb.ClearUserConnOverrideLimitsResponse{
		Cleared: uint32(cleared),
	}, nil
}

func (s *Service) SetUserDefaultPerConnLimit(ctx context.Context, req *ratelimitpb.SetUserDefaultPerConnLimitRequest) (*ratelimitpb.SetUserDefaultPerConnLimitResponse, error) {
	// Минимальная валидация
	if req.Uuid == "" {
		return nil, errors.New("uuid is empty")
	}
	// down/up могут быть 0: тогда это будет “безлимит” или “запрет”? пока трактуем как “0 = безлимит отключён”.
	ratelimit.Limits.SetUserDefault(req.Uuid, req.DownBps, req.UpBps)
	return &ratelimitpb.SetUserDefaultPerConnLimitResponse{}, nil
}

func (s *Service) SetConnectionLimit(ctx context.Context, req *ratelimitpb.SetConnectionLimitRequest) (*ratelimitpb.SetConnectionLimitResponse, error) {
	if req.ConnId == 0 {
		return nil, errors.New("conn_id is 0")
	}
	ratelimit.Limits.SetConnLimit(ratelimit.ConnID(req.ConnId), req.DownBps, req.UpBps)
	return &ratelimitpb.SetConnectionLimitResponse{}, nil
}

func (s *Service) ClearConnectionLimit(ctx context.Context, req *ratelimitpb.ClearConnectionLimitRequest) (*ratelimitpb.ClearConnectionLimitResponse, error) {
	if req.ConnId == 0 {
		return nil, errors.New("conn_id is 0")
	}
	ratelimit.Limits.ClearConnLimit(ratelimit.ConnID(req.ConnId))
	return &ratelimitpb.ClearConnectionLimitResponse{}, nil
}

// ---- wiring через common.CreateObject ----

// New создаёт объект, который Commander добавит в список services
func New(ctx context.Context, cfg *Config) (commander.Service, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.GetKeyMode()))
	switch mode {
	case "uuid":
		ratelimit.SetKeyMode(ratelimit.KeyModeUUID)
	default:
		ratelimit.SetKeyMode(ratelimit.KeyModeDevice)
	}
	return &Service{}, nil
}

func init() {
	errors.LogInfo(context.Background(), "ratelimit/command: init() called")
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		c, ok := cfg.(*Config)
		if !ok {
			return nil, errors.New("invalid config type for ratelimit command")
		}
		return New(ctx, c)
	}))
}
