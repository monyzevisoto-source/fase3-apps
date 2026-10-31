package main

import (
	"context"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newMeterProvider(ctx context.Context, serviceName string, reader metric.Reader) (*metric.MeterProvider, error) {
	if reader == nil {
		exporter, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return nil, err
		}
		reader = metric.NewPeriodicReader(exporter)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
	))
	if err != nil {
		return nil, err
	}

	return metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	), nil
}

func configureMetrics(ctx context.Context, serviceName string) (*metric.MeterProvider, error) {
	provider, err := newMeterProvider(ctx, serviceName, nil)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(provider)
	return provider, nil
}

// configureTraces enables OTLP/HTTP export and W3C propagation across HTTP calls.
func configureTraces(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(attribute.String("service.name", serviceName)))
	if err != nil {
		return nil, err
	}
	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if os.Getenv("OTEL_TRACES_EXPORTER") != "none" {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	} else {
		options = append(options, sdktrace.WithSampler(sdktrace.NeverSample()))
	}
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	return provider, nil
}

func shutdownTraces(provider *sdktrace.TracerProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.Shutdown(ctx); err != nil {
		log.Printf("Erro ao finalizar traces OpenTelemetry: %v", err)
	}
}
