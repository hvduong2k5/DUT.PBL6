package middleware

import "context"

type requestIDKey struct{}

// WithRequestID stores the per-request correlation ID that every error body and
// log line carries.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
