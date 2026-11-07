package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func (a *App) fetchFlag(flagName string) (*Flag, error) {
	return a.fetchFlagContext(context.Background(), flagName)
}

func (a *App) fetchFlagContext(ctx context.Context, flagName string) (*Flag, error) {
	url := fmt.Sprintf("%s/flags/%s", a.FlagServiceURL, flagName)

	apiKey := os.Getenv("SERVICE_API_KEY")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar flag-service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("erro ao fechar resposta HTTP: %v", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{flagName}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("flag-service retornou status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var flag Flag
	if err := json.Unmarshal(body, &flag); err != nil {
		return nil, fmt.Errorf("erro ao desserializar resposta do flag-service: %w", err)
	}
	return &flag, nil
}

// runEvaluationLogic é onde a decisão é tomada
func (a *App) runEvaluationLogic(info *CombinedFlagInfo, userID string) bool {
	if info.Flag == nil || !info.Flag.IsEnabled {
		return false
	}

	if info.Rule == nil || !info.Rule.IsEnabled {
		return true
	}

	// 3. Processa a regra (só temos "PERCENTAGE" por enquanto)
	rule := info.Rule.Rules
	if rule.Type == "PERCENTAGE" {
		// Converte o 'value' (que é interface{}) para float64
		percentage, ok := rule.Value.(float64)
		if !ok {
			log.Printf("Erro: valor da regra de porcentagem não é um número para a flag '%s'", info.Flag.Name)
			return false
		}

		// Calcula o "bucket" do usuário (0-99)
		userBucket := getDeterministicBucket(userID + info.Flag.Name)

		if float64(userBucket) < percentage {
			return true
		}
	}

	return false
}

func getDeterministicBucket(input string) int {
	// Usa SHA-256 e os primeiros 4 bytes para um bucket determinístico.
	hash := sha256.Sum256([]byte(input))

	// Converte 4 bytes para um uint32
	val := binary.BigEndian.Uint32(hash[:4])

	// Retorna o módulo 100
	return int(val % 100)
}
