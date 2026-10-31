package main

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

// redisTraceHook deliberately omits keys, arguments and values.
type redisTraceHook struct{}

func (redisTraceHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	ctx, _ = otel.Tracer("evaluation-service/redis").Start(ctx, strings.ToUpper(cmd.Name()), trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		attribute.String("db.system.name", "redis"), attribute.String("db.system", "redis"), attribute.String("db.operation.name", strings.ToUpper(cmd.Name())),
	))
	return ctx, nil
}
func (redisTraceHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	finishRedisSpan(ctx, cmd.Err())
	return nil
}
func (redisTraceHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	ctx, _ = otel.Tracer("evaluation-service/redis").Start(ctx, "redis pipeline", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("db.system.name", "redis"), attribute.String("db.system", "redis")))
	return ctx, nil
}
func (redisTraceHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	var failure error
	for _, cmd := range cmds {
		if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) {
			failure = cmd.Err()
			break
		}
	}
	finishRedisSpan(ctx, failure)
	return nil
}
func finishRedisSpan(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		span.SetStatus(codes.Error, "Redis command failed")
	}
	span.End()
}
