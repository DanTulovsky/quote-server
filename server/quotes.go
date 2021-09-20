package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

const (
	theySaidSoURL = "http://quotes.rest/qod.json"
)

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

func TheySaidSoQuote(ctx context.Context) string {
	tracer := otel.Tracer("global")
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
	tracer := otel.Tracer("global")
	_, span := tracer.Start(ctx, "getRandomQuote")
	defer span.End()

	span.SetAttributes(attribute.String("quote_source", "static"))

	span.AddEvent("get_quote", trace.WithAttributes(
		attribute.Int("quote.id", 1),
	))

	quote := "some random quote"
	return quote
}
