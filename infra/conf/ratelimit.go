package conf

import (
	"strings"
	"time"

	"github.com/xtls/xray-core/app/ratelimit"
	"github.com/xtls/xray-core/common/errors"
)

type RateLimitConfig struct {
	// "device" или "uuid"
	KeyMode string `json:"keyMode"`
	// optional; if omitted, keep the package default
	GraceSeconds *uint32 `json:"graceSeconds"`
}

func (c *RateLimitConfig) Apply() error {
	mode := strings.ToLower(strings.TrimSpace(c.KeyMode))

	switch mode {
	case "":
		// Keep current mode if config only overrides grace.
	case "device":
		ratelimit.SetKeyMode(ratelimit.KeyModeDevice)
	case "uuid":
		ratelimit.SetKeyMode(ratelimit.KeyModeUUID)
	default:
		return errors.New("unknown ratelimit.keyMode: ", c.KeyMode)
	}

	if c.GraceSeconds != nil {
		ratelimit.SetGrace(time.Duration(*c.GraceSeconds) * time.Second)
	}

	return nil
}
