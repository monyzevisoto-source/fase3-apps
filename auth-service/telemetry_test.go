package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestHTTPMetricsUseAuthServiceNameAndStatus(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider, err := newMeterProvider(ctx, "auth-service", reader)
	if err != nil {
		t.Fatalf("newMeterProvider() error = %v", err)
	}
	otel.SetMeterProvider(provider)

	handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}), "auth-service")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/validate", nil))

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	serviceName := ""
	for _, item := range data.Resource.Attributes() {
		if item.Key == attribute.Key("service.name") {
			serviceName = item.Value.AsString()
		}
	}
	if serviceName != "auth-service" {
		t.Fatalf("service.name = %q, want auth-service", serviceName)
	}
	if !hasHTTPStatus(data, int64(http.StatusUnauthorized)) {
		t.Fatal("HTTP metrics did not contain status 401")
	}
}

func hasHTTPStatus(data metricdata.ResourceMetrics, status int64) bool {
	for _, scope := range data.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if hasStatus(metric.Data, status) {
				return true
			}
		}
	}
	return false
}

func hasStatus(data metricdata.Aggregation, status int64) bool {
	switch histogram := data.(type) {
	case metricdata.Histogram[int64]:
		for _, point := range histogram.DataPoints {
			for _, item := range point.Attributes.ToSlice() {
				if (item.Key == "http.response.status_code" || item.Key == "http.status_code" || item.Key == "http.status.code") && item.Value.AsInt64() == status {
					return true
				}
			}
		}
	case metricdata.Histogram[float64]:
		for _, point := range histogram.DataPoints {
			for _, item := range point.Attributes.ToSlice() {
				if (item.Key == "http.response.status_code" || item.Key == "http.status_code" || item.Key == "http.status.code") && item.Value.AsInt64() == status {
					return true
				}
			}
		}
	}
	return false
}
