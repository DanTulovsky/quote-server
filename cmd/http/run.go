package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/DanTulovsky/quote-server/server"

	"github.com/chenjiandongx/ginprom"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/lightstep/otel-launcher-go/launcher"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelServiceName = "quote"
)

var (
	enableMetrics = flag.Bool("enable_metrics", true, "Set to enable metrics via lightstep (requires tracing is enabled).")
	grpcPort      = flag.String("grpc_port", "8081", "port to server grpc on")
	httpPort      = flag.String("http_port", "8080", "port to server grpc on")
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
	//r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
	//	SkipPaths: []string{"/healthz", "/servez"},
	//}))

	r.GET("/healthz", healthHandler)
	r.GET("/servez", healthHandler)
	// TODO: https://github.com/chenjiandongx/ginprom
	r.GET("/metrics", ginprom.PromHandler(promhttp.Handler()))

	trace := r.Group("/")
	{
		trace.Use(otelgin.Middleware(otelServiceName))
		trace.GET("/", randomQuoteHandler)
	}

	// grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen to grpc: %v", err)
	}
	s := server.NewServer()
	go func() {
		log.Printf("Starting grpc server on port %s", *grpcPort)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve grpc: %v", err)
		}
	}()

	// http server
	log.Printf("Starting http server in port %s", *httpPort)
	endless.ListenAndServe(fmt.Sprintf(":%s", *httpPort), r)
}

func randomQuoteHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// TODO: Write a grpc server that returns the source of the quote
	quote := server.TheySaidSoQuote(ctx)

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

	c.String(http.StatusOK, sb.String())
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
