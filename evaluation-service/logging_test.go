package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	collectorlog "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

func TestStandardLogsExportOverOTLPHTTP(t *testing.T) {
	received := make(chan *collectorlog.ExportLogsServiceRequest, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		req := new(collectorlog.ExportLogsServiceRequest)
		if err := proto.Unmarshal(body, req); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		received <- req
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp")
	original := log.Writer()
	defer log.SetOutput(original)
	provider, err := configureLogs(context.Background(), "logs-integration")
	if err != nil {
		t.Fatal(err)
	}
	defer shutdownLogs(provider)
	log.Print("otlp-logs-test")
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-received:
		if len(req.ResourceLogs) != 1 {
			t.Fatalf("resources: %v", req)
		}
		resource := req.ResourceLogs[0]
		found := false
		for _, a := range resource.Resource.Attributes {
			if a.Key == "service.name" && a.Value.GetStringValue() == "logs-integration" {
				found = true
			}
		}
		if !found {
			t.Fatal("missing service.name")
		}
		records := resource.ScopeLogs[0].LogRecords
		if len(records) != 1 || records[0].Body.GetStringValue() == "" {
			t.Fatalf("logs: %v", records)
		}
	default:
		t.Fatal("no OTLP log request received")
	}
}
