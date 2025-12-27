package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Port               string
	SlackSigningSecret string
	ConfigFile         string
}

// CatalogEntry represents a catalog configuration entry
type CatalogEntry struct {
	ActionID string   `json:"actionId"`
	Options  []Option `json:"options"`
}

// Option represents a single option in the catalog
type Option struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

// SlackRequest represents the incoming Slack request
type SlackRequest struct {
	Type      string `json:"type"`
	ActionID  string `json:"action_id"`
	BlockID   string `json:"block_id"`
	Value     string `json:"value"`
}

// SlackResponse represents the response sent back to Slack
type SlackResponse struct {
	Options []SlackOption `json:"options"`
}

// SlackOption represents a single option in the Slack response
type SlackOption struct {
	Text  SlackText `json:"text"`
	Value string    `json:"value"`
}

// SlackText represents the text field in a Slack option
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

var catalog []CatalogEntry

func main() {
	config := loadConfig()

	if err := loadCatalog(config.ConfigFile); err != nil {
		log.Fatalf("Failed to load catalog: %v", err)
	}

	http.HandleFunc("/", handleRequest(config.SlackSigningSecret))

	log.Printf("Starting server on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable is required")
	}

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "catalog.json"
	}

	return Config{
		Port:               port,
		SlackSigningSecret: signingSecret,
		ConfigFile:         configFile,
	}
}

// loadCatalog loads the catalog from a JSON file
func loadCatalog(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading catalog file: %w", err)
	}

	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("parsing catalog JSON: %w", err)
	}

	log.Printf("Loaded %d catalog entries", len(catalog))
	return nil
}

// handleRequest handles incoming Slack requests
func handleRequest(signingSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Validate Slack signature
		timestamp := r.Header.Get("X-Slack-Request-Timestamp")
		signature := r.Header.Get("X-Slack-Signature")

		if !verifySlackSignature(signingSecret, timestamp, body, signature) {
			log.Printf("Invalid Slack signature")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse the request based on content type
		var slackReq SlackRequest
		contentType := r.Header.Get("Content-Type")
		
		if contentType == "application/x-www-form-urlencoded" {
			// Parse form-encoded data
			values, err := url.ParseQuery(string(body))
			if err != nil {
				log.Printf("Error parsing form data: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			
			// Extract and decode the payload field
			payloadStr := values.Get("payload")
			if payloadStr == "" {
				log.Printf("Missing payload field in form data")
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			
			// Decode JSON from payload
			if err := json.Unmarshal([]byte(payloadStr), &slackReq); err != nil {
				log.Printf("Error parsing payload JSON: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		} else {
			// Handle direct JSON (backward compatibility)
			if err := json.Unmarshal(body, &slackReq); err != nil {
				log.Printf("Error parsing request: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
		}

		log.Printf("Received request for action_id: %s", slackReq.ActionID)

		// Find matching catalog entry
		var options []Option
		for _, entry := range catalog {
			if entry.ActionID == slackReq.ActionID {
				options = entry.Options
				break
			}
		}

		// Build response
		slackOptions := make([]SlackOption, len(options))
		for i, opt := range options {
			slackOptions[i] = SlackOption{
				Text: SlackText{
					Type: "plain_text",
					Text: opt.Text,
				},
				Value: opt.Value,
			}
		}

		response := SlackResponse{
			Options: slackOptions,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}
}

// verifySlackSignature verifies the Slack request signature
func verifySlackSignature(signingSecret, timestamp string, body []byte, signature string) bool {
	// Check timestamp to prevent replay attacks (5 minutes tolerance)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Printf("Invalid timestamp: %v", err)
		return false
	}

	now := time.Now().Unix()
	if abs(now-ts) > 300 {
		log.Printf("Timestamp too old: %d vs %d", ts, now)
		return false
	}

	// Compute expected signature
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBaseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// abs returns the absolute value of an int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
