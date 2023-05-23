package cmd

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type ctxTracerKey struct{}

func TracerFromContext(ctx context.Context) trace.Tracer {
	t, _ := ctx.Value(ctxTracerKey{}).(trace.Tracer)
	return t
}

func NewContextWithTracer(parent context.Context, t trace.Tracer) context.Context {
	return context.WithValue(parent, ctxTracerKey{}, t)
}
