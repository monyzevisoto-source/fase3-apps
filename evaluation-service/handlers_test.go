package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerReturnsOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	(&App{}).healthHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("returned status %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestEvaluationHandlerRequiresParameters(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/evaluate", nil)

	(&App{}).evaluationHandler(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("returned status %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
