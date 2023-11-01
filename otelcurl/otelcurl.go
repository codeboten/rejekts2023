package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	otelEndpoint := "localhost:4317"
	if endpoint, present := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); present {
		otelEndpoint = endpoint
	}

	conn, err := grpc.DialContext(ctx, otelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, err
}

func sendRequest(ctx context.Context, url *string, tr trace.Tracer) error {
	ctx, span := tr.Start(ctx, "roll the dice", trace.WithAttributes(semconv.PeerService("rolldice-server")))
	defer span.End()
	if url == nil || len(*url) == 0 {
		return fmt.Errorf("Must specify an endpoint with -endpoint")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)
	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	fmt.Printf("Sending request...\n")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	var body []byte
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	_ = res.Body.Close()

	fmt.Printf("Response Received: %s\n", body)
	return nil
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	url := flag.String("endpoint", "", "endpoint url")
	interval := flag.Duration("interval", time.Second*10, "interval between requests")
	flag.Parse()

	tracer := otel.Tracer("rejekts/client")

	for {
		err = sendRequest(context.Background(), url, tracer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Waiting %q before next request\n", interval)
		time.Sleep(*interval)
	}
}
