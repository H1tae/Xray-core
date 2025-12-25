package ratelimit

import (
	"context"

	"github.com/xtls/xray-core/common/session"
)

func UUIDFromContext(ctx context.Context) (string, bool) {
	inb := session.InboundFromContext(ctx)
	if inb == nil || inb.User == nil {
		return "", false
	}
	if inb.User.Email == "" {
		// Для нашего MVP требуем, чтобы email был задан и содержал UUID
		return "", false
	}
	return inb.User.Email, true
}
