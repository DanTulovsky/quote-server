module github.com/DanTulovsky/quote-server

go 1.16

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/lightstep/otel-launcher-go v0.18.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.18.0
	go.opentelemetry.io/otel v0.18.0
	go.opentelemetry.io/otel/trace v0.18.0
)
