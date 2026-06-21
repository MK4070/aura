package telemetry

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	apiMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

var (
	ChunksProcessed   apiMetric.Int64Counter
	IngestionDuration apiMetric.Float64Histogram
)

func InitProvider() func(context.Context) error {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to initialize metric exporter: %v", err)
	}

	reader := metric.NewPeriodicReader(exporter)
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	meter := provider.Meter("aura/ingestion")

	ChunksProcessed, _ = meter.Int64Counter(
		"chunks_processed_total",
		apiMetric.WithDescription("Total document chunks routed to Qdrant"),
	)

	IngestionDuration, _ = meter.Float64Histogram(
		"ingestion_duration_seconds",
		apiMetric.WithDescription("Time taken to chunk, embed, and push a file to Qdrant"),
		apiMetric.WithUnit("s"),
	)

	return provider.Shutdown
}
