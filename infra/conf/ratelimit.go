package conf

import (
	"strings"

	"github.com/xtls/xray-core/app/ratelimit"
	"github.com/xtls/xray-core/common/errors"
)

type RateLimitConfig struct {
	// "device" или "uuid"
	KeyMode string `json:"keyMode"`
}

func (c *RateLimitConfig) Apply() error {
	mode := strings.ToLower(strings.TrimSpace(c.KeyMode))

	switch mode {
	case "device":
		ratelimit.SetKeyMode(ratelimit.KeyModeDevice)
		return nil
	case "", "uuid":
		ratelimit.SetKeyMode(ratelimit.KeyModeUUID)
		return nil
	default:
		return errors.New("unknown ratelimit.keyMode: ", c.KeyMode)
	}
}
