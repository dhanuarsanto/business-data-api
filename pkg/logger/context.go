package logger

import "context"

type contextKey string

const TraceCtxKey contextKey = "traceCtx"

type TraceContext struct {
	TraceID   string
	Developer string
	IP        string
	Path      string
	UserID    int
}

func SetTraceContext(ctx context.Context, tCtx *TraceContext) context.Context {
	return context.WithValue(ctx, TraceCtxKey, tCtx)
}

func GetTraceContext(ctx context.Context) *TraceContext {
	if tCtx, ok := ctx.Value(TraceCtxKey).(*TraceContext); ok {
		return tCtx
	}
	return &TraceContext{
		TraceID:   "UNKNOWN",
		Developer: "UNKNOWN",
		IP:        "UNKNOWN",
		Path:      "UNKNOWN",
	}
}
