package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type traceTestDriver struct{}
type traceTestConn struct{}
type traceTestRows struct{ done bool }

func (traceTestDriver) Open(string) (driver.Conn, error)  { return traceTestConn{}, nil }
func (traceTestConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unsupported") }
func (traceTestConn) Close() error                        { return nil }
func (traceTestConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }
func (traceTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query == "failure" {
		return nil, errors.New("private database error")
	}
	return &traceTestRows{done: query == "missing"}, nil
}
func (*traceTestRows) Columns() []string { return []string{"id"} }
func (*traceTestRows) Close() error      { return nil }
func (r *traceTestRows) Next(values []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	values[0] = int64(1)
	return nil
}
func init() { sql.Register("trace-test", traceTestDriver{}) }

func TestPostgresQuerySpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)
	defer provider.Shutdown(context.Background())
	db, err := sql.Open("trace-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := &App{DB: db}
	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	defer parent.End()
	for _, query := range []string{"success", "missing", "failure"} {
		var id int
		err := app.queryKeyRow(ctx, "SELECT", query, []any{"secret-api-key"}, &id)
		if query == "success" && (err != nil || id != 1) {
			t.Fatalf("query failed: %v", err)
		}
		if query == "missing" && !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected no rows, got %v", err)
		}
	}
	spans := recorder.Ended()
	if len(spans) != 3 {
		t.Fatalf("got %d spans", len(spans))
	}
	for i, span := range spans {
		if span.Parent().SpanID() != parent.SpanContext().SpanID() || span.SpanKind() != trace.SpanKindClient {
			t.Error("lost PostgreSQL parent or client kind")
		}
		want := codes.Unset
		if i == 2 {
			want = codes.Error
		}
		if span.Status().Code != want {
			t.Errorf("status %v, want %v", span.Status().Code, want)
		}
		for _, attr := range span.Attributes() {
			if strings.Contains(attr.Value.AsString(), "secret") {
				t.Error("SQL parameter leaked")
			}
		}
	}
}
