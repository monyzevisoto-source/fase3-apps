package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func configureLogs(ctx context.Context, serviceName string) (*sdklog.LoggerProvider, error) {
	if os.Getenv("OTEL_LOGS_EXPORTER") == "none" {
		return sdklog.NewLoggerProvider(), nil
	}
	exporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, err
	}
	res := resource.NewSchemaless(attribute.String("service.name", serviceName))
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	handler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(provider))
	bridge := slog.NewLogLogger(handler, slog.LevelInfo)
	log.SetOutput(io.MultiWriter(os.Stderr, bridge.Writer()))
	// Exporter failures use the default OTel error handler (stderr), not log.Print.
	return provider, nil
}

func shutdownLogs(provider *sdklog.LoggerProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = provider.Shutdown(ctx)
}
