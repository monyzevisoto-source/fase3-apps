package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForcedFailureHandlerAlwaysReturns500(t *testing.T) {
	for _, path := range []string{"/health", "/evaluate", "/unknown"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)

		forcedFailureHandler(recorder, request)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("path %s returned status %d, want %d", path, recorder.Code, http.StatusInternalServerError)
		}
	}
}
