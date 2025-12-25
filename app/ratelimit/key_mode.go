package ratelimit

import "sync/atomic"

type KeyMode int32

const (
	KeyModeUUID   KeyMode = 0 // uuid only
	KeyModeDevice KeyMode = 1 // uuid+srcIP
)

var keyMode atomic.Int32

func init() {
	keyMode.Store(int32(KeyModeDevice)) // дефолт — как было
}

func SetKeyMode(m KeyMode) { keyMode.Store(int32(m)) }
func GetKeyMode() KeyMode  { return KeyMode(keyMode.Load()) }

func BuildDeviceKey(uuid string, srcIP string) string {
	switch GetKeyMode() {
	case KeyModeDevice:
		return uuid + "|" + srcIP
	default: // KeyModeUUID
		return uuid
	}
}
