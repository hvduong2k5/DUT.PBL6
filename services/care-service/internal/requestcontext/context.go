package requestcontext

import "context"

type requestIDKey struct{}
type correlationIDKey struct{}

func WithIDs(ctx context.Context, requestID, correlationID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey{}, requestID)
	return context.WithValue(ctx, correlationIDKey{}, correlationID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func CorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey{}).(string)
	return value
}
