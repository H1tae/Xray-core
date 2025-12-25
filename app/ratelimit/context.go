package ratelimit

import "context"

type ctxKey struct{}

func WithInboundConnID(ctx context.Context, id ConnID) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func InboundConnIDFromContext(ctx context.Context) (ConnID, bool) {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return 0, false
	}
	id, ok := v.(ConnID)
	return id, ok
}
