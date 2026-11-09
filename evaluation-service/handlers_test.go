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

func TestErrorModeMakesEvaluationReturnInternalServerError(t *testing.T) {
	app := &App{}

	activateRecorder := httptest.NewRecorder()
	activateRequest := httptest.NewRequest(http.MethodPost, "/error", nil)
	app.errorHandler(activateRecorder, activateRequest)

	if activateRecorder.Code != http.StatusOK {
		t.Fatalf("activation returned status %d, want %d", activateRecorder.Code, http.StatusOK)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/evaluate?user_id=user-1&flag_name=flag-1", nil)
	app.evaluationHandler(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("error mode returned status %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestHealthRemainsAvailableInErrorMode(t *testing.T) {
	app := &App{}
	app.errorMode.Store(true)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	app.healthHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health returned status %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestDisableErrorModeRestoresEvaluationHandler(t *testing.T) {
	app := &App{}
	app.errorMode.Store(true)

	disableRecorder := httptest.NewRecorder()
	disableRequest := httptest.NewRequest(http.MethodPost, "/error/disable", nil)
	app.disableErrorHandler(disableRecorder, disableRequest)

	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("deactivation returned status %d, want %d", disableRecorder.Code, http.StatusOK)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/evaluate", nil)
	app.evaluationHandler(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("evaluation after deactivation returned status %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
