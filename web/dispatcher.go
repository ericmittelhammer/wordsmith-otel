package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func newExporter(ctx context.Context) (*otlptrace.Exporter, error) {

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	return exporter, err

}

func newTraceProvider(exp sdktrace.SpanExporter, serviceName string) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func main() {

	serviceName := os.Getenv("OTEL_SERVICE_NAME")

	rand.Seed(time.Now().UnixNano())

	ctx := context.Background()

	exp, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider(exp, serviceName)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	tracer = tp.Tracer("nr-otel-demos/wordsmith-otel")

	fwd := &forwarder{"api", 8080}
	http.Handle("/words/", otelhttp.NewHandler(http.StripPrefix("/words", fwd), "forwarder"))
	http.Handle("/", otelhttp.NewHandler(http.FileServer(http.Dir("static")), "static"))

	fmt.Println("Listening on port 80")
	http.ListenAndServe(":80", nil)
}

type forwarder struct {
	host string
	port int
}

func (f *forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addrs, err := net.LookupHost(f.host)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, err.Error(), 500)
		return
	}

	log.Printf("%s %d available ips: %v", r.URL.Path, len(addrs), addrs)
	ip := addrs[rand.Intn(len(addrs))]
	log.Printf("%s I choose %s", r.URL.Path, ip)

	url := fmt.Sprintf("http://%s:%d%s", ip, f.port, r.URL.Path)
	log.Printf("%s Calling %s", r.URL.Path, url)

	if err = copy(url, ip, w); err != nil {
		log.Println("Error", err)
		http.Error(w, err.Error(), 500)
		return
	}
}

func copy(url, ip string, w http.ResponseWriter) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.Header().Set("source", ip)

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	_, err = w.Write(buf)
	return err
}
