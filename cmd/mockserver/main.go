// mockserver is a development/test tool that pretends to be api.anthropic.com.
// It logs request bodies to stdout so you can verify PII was sanitized before
// reaching the "AI server".
//
// Usage:
//
//	go run cmd/mockserver/main.go
//
// Then in another terminal:
//
//	saola proxy --port 8080
//
// And test with curl:
//
//	curl -x http://localhost:8080 \
//	  -d '{"messages":[{"content":"my email is test@secret.com"}]}' \
//	  http://localhost:9090/v1/messages
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", handleMessages)
	mux.HandleFunc("/", handleCatchAll)

	addr := ":9090"
	log.Printf("mockserver: listening on %s (pretending to be api.anthropic.com)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("mockserver: %v", err)
	}
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[%s] POST /v1/messages — body: %s", time.Now().Format(time.RFC3339), string(body))

	// Return a simple Anthropic-style response that echoes back a placeholder
	// so rehydration can also be tested.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
  "id": "mock-msg-001",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "I received your message."}],
  "model": "claude-mock",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 8}
}`)
}

func handleCatchAll(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[%s] %s %s — body: %s", time.Now().Format(time.RFC3339), r.Method, r.URL.Path, string(body))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status":"ok"}`)
}
