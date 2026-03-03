package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "product-catalog-service"

// EndFunc ends a span, recording an error if non-nil.
type EndFunc func(err error)

// StartSpan creates a child span and returns a function to end it.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, EndFunc) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, name,
		trace.WithAttributes(attrs...),
	)
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}
