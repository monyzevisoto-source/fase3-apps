package main

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestTracesExportAndPropagation(t *testing.T) {
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	defer otel.SetTracerProvider(previousProvider)
	defer otel.SetTextMapPropagator(previousPropagator)
	received := make(chan *collector.ExportTraceServiceRequest, 1)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" || r.Header.Get("api-key") != "test-key" {
			t.Errorf("unexpected OTLP path or auth")
		}
		body, _ := io.ReadAll(r.Body)
		request := &collector.ExportTraceServiceRequest{}
		if err := proto.Unmarshal(body, request); err != nil {
			t.Error(err)
		}
		received <- request
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer endpoint.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", endpoint.URL+"/v1/traces")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "api-key=test-key")
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	provider, err := configureTraces(context.Background(), "test-service")
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownTraces(provider)
	handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer("test").Start(r.Context(), "child")
		span.End()
	}), "incoming")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case exported := <-received:
		count := 0
		for _, resource := range exported.ResourceSpans {
			for _, scope := range resource.ScopeSpans {
				for _, span := range scope.Spans {
					count++
					if got := hex.EncodeToString(span.TraceId); got != "0123456789abcdef0123456789abcdef" {
						t.Errorf("trace continuity lost: %s", got)
					}
				}
			}
		}
		if count != 2 {
			t.Errorf("expected server and child spans, got %d", count)
		}
	default:
		t.Fatal("no spans exported")
	}
}
