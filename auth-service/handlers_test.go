package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&App{}).healthHandler(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestMasterKeyAuthMiddleware(t *testing.T) {
	app := &App{MasterKey: "secret"}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := app.masterKeyAuthMiddleware(next)

	for _, test := range []struct {
		name       string
		authorize  string
		wantStatus int
	}{
		{name: "missing key", authorize: "", wantStatus: http.StatusForbidden},
		{name: "wrong key", authorize: "Bearer wrong", wantStatus: http.StatusForbidden},
		{name: "valid key", authorize: "Bearer secret", wantStatus: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/admin/keys", nil)
			if test.authorize != "" {
				request.Header.Set("Authorization", test.authorize)
			}
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}
