package main

import (
	"net/http"
)

type EvaluationResponse struct {
	FlagName string `json:"flag_name"`
	UserID   string `json:"user_id"`
	Result   bool   `json:"result"`
}

// forcedFailureHandler intentionally makes every HTTP request fail. This is
// used only to simulate a bad deployment and exercise automatic rollback.
func forcedFailureHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"simulated deployment failure"}`, http.StatusInternalServerError)
}
