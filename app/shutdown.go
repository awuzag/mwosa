package app

import (
	"context"
	"time"
)

const runtimeShutdownTimeout = 10 * time.Second

func runtimeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func shutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, runtimeShutdownTimeout)
}
