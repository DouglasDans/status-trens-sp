package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// functionURL adapts an http.Handler to a Lambda Function URL, which hands
// the function an event instead of an HTTP request. The responses here are
// small JSON documents, so they are buffered and returned as text.
func functionURL(h http.Handler) func(context.Context, events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return func(ctx context.Context, e events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		body := e.Body
		if e.IsBase64Encoded {
			decoded, err := base64.StdEncoding.DecodeString(e.Body)
			if err != nil {
				return events.LambdaFunctionURLResponse{}, fmt.Errorf("decoding body: %w", err)
			}
			body = string(decoded)
		}
		target := e.RawPath
		if e.RawQueryString != "" {
			target += "?" + e.RawQueryString
		}
		req := httptest.NewRequestWithContext(ctx, e.RequestContext.HTTP.Method, target, strings.NewReader(body))
		for k, v := range e.Headers {
			req.Header.Set(k, v)
		}

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		headers := map[string]string{}
		for k, v := range rec.Header() {
			headers[k] = strings.Join(v, ",")
		}
		return events.LambdaFunctionURLResponse{StatusCode: rec.Code, Headers: headers, Body: rec.Body.String()}, nil
	}
}
