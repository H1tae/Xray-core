package ratelimit

import (
	"context"
	"sync/atomic"
)

type ChooseOutboundFunc func(ctx context.Context, uuid string) (tag string, ok bool, err error)

var chooser atomic.Value // stores ChooseOutboundFunc

// DefaultChooser: заглушка.
// Возвращает ok=false => routedDispatch НЕ будет форсить outbound,
// и дальше сработает routing (если включён) или default outbound.
func DefaultChooser(ctx context.Context, uuid string) (string, bool, error) {
	return "", false, nil
}

func SetOutboundChooser(f ChooseOutboundFunc) {
	if f == nil {
		f = DefaultChooser
	}
	chooser.Store(f)
}

func chooseRaw(ctx context.Context, uuid string) (string, bool, error) {
	if v := chooser.Load(); v != nil {
		return v.(ChooseOutboundFunc)(ctx, uuid)
	}
	// на всякий случай, если init ещё не успел
	return DefaultChooser(ctx, uuid)
}

// Выбор outbound c учётом deviceKey:
// 1) смотрим deviceEntry.egressTag
// 2) если нет — вызываем raw chooser, сохраняем в deviceEntry.egressTag
func ChooseOutboundTagForDevice(ctx context.Context, uuid string, deviceKey string) (string, bool, error) {
	if tag, ok := DeviceGetEgress(deviceKey); ok && tag != "" {
		return tag, true, nil
	}

	tag, ok, err := chooseRaw(ctx, uuid)
	if err != nil || !ok || tag == "" {
		return "", false, err
	}

	DeviceSetEgress(deviceKey, tag)
	return tag, true, nil
}

func init() {
	SetOutboundChooser(DefaultChooser)
}
