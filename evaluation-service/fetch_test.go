package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchFlagReturnsNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	app := &App{HttpClient: server.Client(), FlagServiceURL: server.URL}
	_, err := app.fetchFlag("missing")
	if _, ok := err.(*NotFoundError); !ok {
		t.Fatalf("error type = %T, want *NotFoundError", err)
	}
}

func TestFetchFlagDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing service authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"checkout","is_enabled":true}`))
	}))
	defer server.Close()

	t.Setenv("SERVICE_API_KEY", "test-key")
	app := &App{HttpClient: server.Client(), FlagServiceURL: server.URL}
	flag, err := app.fetchFlag("checkout")
	if err != nil {
		t.Fatalf("fetchFlag() returned error: %v", err)
	}
	if flag.Name != "checkout" || !flag.IsEnabled {
		t.Fatalf("decoded flag = %+v", flag)
	}
}
