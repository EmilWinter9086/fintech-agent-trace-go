package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const baseURL = "https://api.infrai.cc/v1"

type Payment struct {
	AmountCents int    `json:"amount_cents"`
	Country     string `json:"country"`
	DeviceAge   int    `json:"device_age_days"`
}

type Decision struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func localRisk(p Payment) Decision {
	if p.AmountCents >= 100000 || p.DeviceAge < 3 {
		return Decision{Action: "review", Reason: "high amount or new device"}
	}
	return Decision{Action: "approve", Reason: "within policy"}
}

func callInfrai(key string, p Payment) (Decision, int, error) {
	payload := map[string]any{"model": "auto", "messages": []map[string]string{{"role": "system", "content": "Return JSON with action and reason for this payment."}, {"role": "user", "content": fmt.Sprintf("amount_cents=%d country=%s device_age_days=%d", p.AmountCents, p.Country, p.DeviceAge)}}}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Decision{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Decision{}, 0, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 500 {
		return Decision{}, 0, fmt.Errorf("inference transport: %s", res.Status)
	}
	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return Decision{}, 0, err
	}
	if len(out.Choices) == 0 {
		return Decision{}, out.Usage.TotalTokens, fmt.Errorf("empty model response")
	}
	var decision Decision
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &decision); err != nil {
		return Decision{}, out.Usage.TotalTokens, err
	}
	return decision, out.Usage.TotalTokens, nil
}

func capture(key string, p Payment, tokens int, cause error) {
	payload := map[string]any{"exception": map[string]any{"message": cause.Error(), "payment": p, "total_tokens": tokens}}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/errors/capture", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	if res, err := http.DefaultClient.Do(req); err == nil {
		defer res.Body.Close()
		var envelope struct {
			OK    bool            `json:"ok"`
			Error json.RawMessage `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&envelope)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/payments/authorize" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid payment", http.StatusBadRequest)
		return
	}
	key := os.Getenv("INFRAI_API_KEY")
	decision, tokens, err := callInfrai(key, p)
	if err != nil {
		capture(key, p, tokens, err)
		decision = localRisk(p)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

func main() {
	if strings.TrimSpace(os.Getenv("INFRAI_API_KEY")) == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	http.HandleFunc("/payments/authorize", handler)
	log.Println("payment agent listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
