package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type EvaluationResponse struct {
	FlagName string `json:"flag_name"`
	UserID   string `json:"user_id"`
	Result   bool   `json:"result"`
}

func (a *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Erro ao serializar resposta de saúde: %v", err)
	}
}

// errorHandler ativa um modo de falha controlado para testar alertas e auto-healing.
// A probe de saúde permanece disponível para que o pod não seja reiniciado pelo Kubernetes.
func (a *App) errorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"método não permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	a.errorMode.Store(true)
	writeJSON(w, http.StatusOK, map[string]string{"status": "error mode enabled"})
}

func (a *App) disableErrorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, `{"error":"método não permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	a.errorMode.Store(false)
	writeJSON(w, http.StatusOK, map[string]string{"status": "error mode disabled"})
}

func (a *App) evaluationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if a.errorMode.Load() {
		http.Error(w, `{"error":"modo de erro habilitado"}`, http.StatusInternalServerError)
		return
	}

	userID := r.URL.Query().Get("user_id")
	flagName := r.URL.Query().Get("flag_name")
	if userID == "" || flagName == "" {
		http.Error(w, `{"error": "user_id e flag_name são obrigatórios"}`, http.StatusBadRequest)
		return
	}

	result, err := a.getDecisionContext(r.Context(), userID, flagName)
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			result = false
		} else {
			log.Printf("Erro ao avaliar flag '%s': %v", flagName, err)
			http.Error(w, `{"error": "Erro interno ao avaliar a flag"}`, http.StatusBadGateway)
			return
		}
	}

	a.eventWorkers.Add(1)
	go func() {
		defer a.eventWorkers.Done()
		eventCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()
		a.sendEvaluationEventContext(eventCtx, userID, flagName, result)
	}()

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(EvaluationResponse{
		FlagName: flagName,
		UserID:   userID,
		Result:   result,
	}); err != nil {
		log.Printf("Erro ao serializar resposta de avaliação: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Erro ao serializar resposta JSON: %v", err)
	}
}
