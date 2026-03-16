/*
Toy OpenTelemetry example for a Go application.

What this shows:
- Traces: creates one span per HTTP request.
- Metrics: increments a request counter per endpoint.
- Logs: emits OTLP logs to the collector.

Prerequisites:
1) Go 1.22+
2) From this folder, initialize and fetch dependencies:
    go mod init examples/go-otel-toy
    go get go.opentelemetry.io/otel@latest
    go get go.opentelemetry.io/otel/sdk@latest
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
    go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@latest
    go get go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc@latest
    go run go_app_otel_example.go

Collector configuration:
- Uses OTLP gRPC endpoint from OTEL_EXPORTER_OTLP_ENDPOINT.
- Defaults to: opentelemetry-collector.opentelemetry.svc.cluster.local:4317
- Basic authentication:
    - OPENTELEMETRY_USERNAME
    - OPENTELEMETRY_PASSWORD
*/
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type otelRuntime struct {
	tp      *sdktrace.TracerProvider
	mp      *sdkmetric.MeterProvider
	lp      *sdklog.LoggerProvider
	tracer  trace.Tracer
	counter metric.Int64Counter
	logger  otellog.Logger
}

func main() {
	ctx := context.Background()
	runtime, err := setupOTel(ctx)
	if err != nil {
		log.Fatalf("failed to setup OpenTelemetry: %v", err)
	}
	defer func() {
		_ = runtime.lp.Shutdown(ctx)
		_ = runtime.mp.Shutdown(ctx)
		_ = runtime.tp.Shutdown(ctx)
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := runtime.tracer.Start(r.Context(), "GET /")
		defer span.End()

		runtime.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("http.route", "/")))

		record := otellog.Record{}
		record.SetTimestamp(time.Now())
		record.SetBody(otellog.StringValue("Handled Go request"))
		record.AddAttributes(
			otellog.String("http.route", "/"),
			otellog.String("http.method", r.Method),
		)
		runtime.logger.Emit(ctx, record)

		fmt.Fprintln(w, "Hello from Go + OpenTelemetry toy example")
	})

	log.Println("listening on http://127.0.0.1:8081")
	if err := http.ListenAndServe("127.0.0.1:8081", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func setupOTel(ctx context.Context) (*otelRuntime, error) {
	endpoint := envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "opentelemetry-collector.opentelemetry.svc.cluster.local:4317")
	serviceName := envOr("OTEL_SERVICE_NAME", "go-toy-app")
	headers := otlpHeaders()

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	logExporter, err := otlploggrpc.New(
		ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	tracer := tp.Tracer("examples.go.app")
	meter := mp.Meter("examples.go.app")
	counter, err := meter.Int64Counter("toy_http_requests_total")
	if err != nil {
		return nil, err
	}
	logger := lp.Logger("examples.go.app")

	return &otelRuntime{
		tp:      tp,
		mp:      mp,
		lp:      lp,
		tracer:  tracer,
		counter: counter,
		logger:  logger,
	}, nil
}

func envOr(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func otlpHeaders() map[string]string {
	username := os.Getenv("OPENTELEMETRY_USERNAME")
	password := os.Getenv("OPENTELEMETRY_PASSWORD")
	token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return map[string]string{"authorization": "Basic " + token}
}
