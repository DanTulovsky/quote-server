package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/chenjiandongx/ginprom"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/lightstep/otel-launcher-go/launcher"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelServiceName = "quote"
	theySaidSoURL   = "http://quotes.rest/qod.json"
)

var (
	enableMetrics = flag.Bool("enable_metrics", true, "Set to enable metrics via lightstep (requires tracing is enabled).")
	version       = flag.String("version", "", "version of the binary")
	tracer        trace.Tracer
)

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting version: %v", *version)

	ls := enableOpenTelemetry()
	tracer = otel.Tracer("global")
	defer ls.Shutdown()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ginprom.PromMiddleware(nil))
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthz", "/servez"},
	}))

	r.GET("/healthz", healthHandler)
	r.GET("/servez", healthHandler)
	// TODO: https://github.com/chenjiandongx/ginprom
	r.GET("/metrics", ginprom.PromHandler(promhttp.Handler()))

	trace := r.Group("/")
	{
		trace.Use(otelgin.Middleware(otelServiceName))
		trace.GET("/", randomQuoteHandler)
	}

	endless.ListenAndServe(":8080", r)
	// r.Run()
}

func randomQuoteHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// TODO: Write a grpc server that returns the source of the quote
	quote := theysaidsoQuote(ctx)

	header := "<html><body>"
	footer := "</body></html>"
	attribution := `
	<span style="z-index:50;font-size:0.9em; font-weight: bold;">
        <img src="https://theysaidso.com/branding/theysaidso.png" height="20" width="20" alt="theysaidso.com"/>
        <a href="https://theysaidso.com" title="Powered by quotes from theysaidso.com" style="color: #ccc; margin-left: 4px; vertical-align: middle;">
        They Said So®
        </a>
    </span>
	`

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("<div>" + quote + "</div>")
	sb.WriteString(attribution)
	sb.WriteString(footer)

	c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	//c.Header("Permissions-Policy", "")
	c.String(http.StatusOK, sb.String())
}

type QuotaSearchError struct {
	Error *qsError `json:"error"`
}

type qsError struct {
	Code    float64 `json:"code"`
	Message string  `json:"message"`
}
type QuoteSearchResult struct {
	Status    *Total     `json:"success"`
	Contents  *QuoteList `json:"contents"`
	BaseURL   string     `json:"baseurl"`
	Copyright *Copyright `json:"copyright"`
}

type Copyright struct {
	Year float64 `json:"year"`
	URL  string  `json:"url"`
}

type Total struct {
	Total int `json:"total"`
}

type QuoteList struct {
	Quotes []Quote `json:"quotes"`
}

type Quote struct {
	Quote      string   `json:"quote"`
	Author     string   `json:"author"`
	Length     string   `json:"length"`
	Tags       []string `json:"tags"`
	Category   string   `json:"category"`
	Language   string   `json:"language"`
	Title      string   `json:"title"`
	Date       string   `json:"date"`
	Id         string   `json:"id"`
	Background string   `json:"background"`
	Permalink  string   `json:"permalink"`
}

func theysaidsoQuote(ctx context.Context) string {
	_, span := tracer.Start(ctx, "theysaidsoQuote",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethodKey.String("GET"),
			semconv.HTTPURLKey.String(theySaidSoURL),
		),
	)
	defer span.End()

	span.SetAttributes(attribute.String("quote_source", theySaidSoURL))

	httpClient := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", theySaidSoURL, nil)
	if err != nil {
		return ""
	}

	span.AddEvent("Retrieving quote")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	span.AddEvent("Retrieved quote")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}
	span.SetAttributes(semconv.HTTPStatusCodeKey.Int(resp.StatusCode))

	// body, err := io.ReadAll(resp.Body)
	// log.Println(string(body))

	var result QuoteSearchResult
	if err := json.Unmarshal(body, &result); err != nil || result.Contents == nil {
		// could be due to error from server
		var errResult QuotaSearchError
		if err := json.Unmarshal(body, &errResult); err != nil || errResult.Error == nil {
			span.RecordError(err)
			return fmt.Sprintf("failed to decode response from server: %v", err)
		}
		span.RecordError(fmt.Errorf("%v", errResult.Error.Message))
		return errResult.Error.Message
	}

	if len(result.Contents.Quotes) > 0 {
		return result.Contents.Quotes[0].Quote
	}

	span.RecordError(errors.New("did not receive any quotes from server"))
	return fmt.Sprint("did not receive any quotes from server")
}

func staticQuote(ctx context.Context) string {
	// otelgin Middleware puts trace ID into context if available
	_, span := tracer.Start(ctx, "staticQuote")
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
