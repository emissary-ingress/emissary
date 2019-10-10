package middleware

import (
	"context"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func WithLogger(ctx context.Context, logger types.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func GetLogger(ctx context.Context) types.Logger {
	return ctx.Value(loggerContextKey{}).(types.Logger)
}

type loggerContextKey struct{}
