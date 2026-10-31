package main

import (
	"context"
	"database/sql"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// queryKeyRow traces the complete query and scan without exporting SQL parameters.
func (a *App) queryKeyRow(ctx context.Context, operation, query string, args []any, dest ...any) error {
	ctx, span := otel.Tracer("auth-service/postgresql").Start(ctx, operation+" api_keys", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"), attribute.String("db.system", "postgresql"),
		attribute.String("db.operation.name", operation), attribute.String("db.collection.name", "api_keys"),
	))
	defer span.End()
	err := a.DB.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		span.SetStatus(codes.Error, "PostgreSQL query failed")
	}
	return err
}
