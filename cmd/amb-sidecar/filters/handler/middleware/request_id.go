package middleware

import (
	"context"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func GetRequestID(ctx context.Context) string {
	return ctx.Value(requestIDContextKey{}).(string)
}

type requestIDContextKey struct{}
