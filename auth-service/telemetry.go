package main

import (
	"context"

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
