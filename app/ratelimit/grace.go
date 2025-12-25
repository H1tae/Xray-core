package ratelimit

import (
	"sync/atomic"
	"time"
)

var graceNs atomic.Int64

func init() {
	graceNs.Store(int64((10 * time.Second).Nanoseconds()))
}

func GetGrace() time.Duration {
	return time.Duration(graceNs.Load())
}

func SetGrace(d time.Duration) {
	if d < 0 {
		d = 0
	}
	graceNs.Store(int64(d))
}
