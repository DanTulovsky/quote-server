package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lightstep/otel-launcher-go/launcher"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelServiceName = "quote"
)

var (
	enableMetrics = flag.Bool("enable_metrics", true, "Set to enable metrics via lightstep (requires tracing is enabled).")
	version       = flag.String("version", "", "version of the binary")
	tracer        trace.Tracer
)

func main() {
	flag.Parse()
	log.Printf("Starting version: %v", *version)

	ls := enableOpenTelemetry()
	tracer = otel.Tracer("global")
	defer ls.Shutdown()

	r := gin.Default()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthz", "/servez"},
	}))

	r.GET("/healthz", healthHandler)
	r.GET("/servez", healthHandler)

	trace := r.Group("/")
	trace.Use(otelgin.Middleware(otelServiceName))
	trace.GET("/", randomQuoteHandler)

	r.Run()
}

func randomQuoteHandler(c *gin.Context) {
	ctx := c.Request.Context()

	c.JSON(http.StatusOK, gin.H{
		"quote": randomQuote(ctx),
	})
}

func randomQuote(ctx context.Context) string {
	// otelgin Middleware puts trace ID into context if available
	_, span := tracer.Start(ctx, "getRandomQuote")
	defer span.End()

	span.SetAttributes(attribute.String("quote_source", "static"))

	span.AddEvent("get_quote", trace.WithAttributes(
		attribute.Int("quote.id", 1),
	))

	quote := "some random quote"
	return quote
}

func healthHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func enableOpenTelemetry() launcher.Launcher {
	log.Println("Enabling OpenTelemetry support...")
	// https://github.com/lightstep/otel-launcher-go
	ls := launcher.ConfigureOpentelemetry(
		launcher.WithServiceName(otelServiceName),
		launcher.WithServiceVersion(*version),
		// launcher.WithAccessToken("{your_access_token}"),  # in env
		launcher.WithLogLevel("info"),
		// launcher.WithPropagators([]string{"b3", "tracecontext"}),
		launcher.WithPropagators([]string{"b3", "baggage", "tracecontext"}),
		launcher.WithMetricsEnabled(*enableMetrics),
	)
	return ls
}
