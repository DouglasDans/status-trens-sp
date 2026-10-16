package main

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func urlRequest(method, path, query string) events.LambdaFunctionURLRequest {
	return events.LambdaFunctionURLRequest{
		RawPath:        path,
		RawQueryString: query,
		Headers:        map[string]string{"accept": "application/json"},
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{Method: method},
		},
	}
}

func TestFunctionURLServesTheHandlerResponse(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, `{"error":"no data"}`)
	})

	res, err := functionURL(h)(context.Background(), urlRequest("GET", "/status", ""))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusServiceUnavailable || res.Body != `{"error":"no data"}` {
		t.Errorf("response = %d %q, want 503 with the handler body", res.StatusCode, res.Body)
	}
	if res.Headers["Content-Type"] != "application/json" {
		t.Errorf("headers = %v, want the handler's Content-Type", res.Headers)
	}
}

func TestFunctionURLPassesTheRequestThrough(t *testing.T) {
	var got *http.Request
	var body string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	})
	req := urlRequest("POST", "/status", "a=1&b=2")
	req.Body = base64.StdEncoding.EncodeToString([]byte("hello"))
	req.IsBase64Encoded = true

	if _, err := functionURL(h)(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got.Method != "POST" || got.URL.Path != "/status" || got.URL.Query().Get("b") != "2" {
		t.Errorf("request = %s %s, want POST /status?a=1&b=2", got.Method, got.URL)
	}
	if got.Header.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q, want the event header", got.Header.Get("Accept"))
	}
	if body != "hello" {
		t.Errorf("body = %q, want the decoded event body", body)
	}
}

func TestFunctionURLRejectsABrokenBody(t *testing.T) {
	req := urlRequest("POST", "/status", "")
	req.Body = "not base64!"
	req.IsBase64Encoded = true

	if _, err := functionURL(http.NotFoundHandler())(context.Background(), req); err == nil {
		t.Error("expected an error for a body that is not base64")
	}
}
