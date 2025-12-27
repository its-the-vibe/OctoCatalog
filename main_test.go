package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// setupTestCatalog initializes a test catalog
func setupTestCatalog() {
	catalog = []CatalogEntry{
		{
			ActionID: "test_action",
			Options: []Option{
				{Text: "Option 1", Value: "opt1"},
				{Text: "Option 2", Value: "opt2"},
			},
		},
	}
}

// generateTestSignature generates a valid Slack signature for testing
func generateTestSignature(secret, timestamp string, body []byte) string {
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBaseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHandleRequest_FormEncoded(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create a Slack request
	slackReq := SlackRequest{
		Type:     "block_suggestion",
		ActionID: "test_action",
		BlockID:  "test_block",
		Value:    "test",
	}

	// Convert to JSON
	jsonPayload, err := json.Marshal(slackReq)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Create form-encoded body with payload field
	formData := url.Values{}
	formData.Set("payload", string(jsonPayload))
	body := formData.Encode()

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add Slack signature headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := generateTestSignature(secret, timestamp, []byte(body))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := handleRequest(secret)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response SlackResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify options
	if len(response.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(response.Options))
	}

	if response.Options[0].Text.Text != "Option 1" {
		t.Errorf("Expected first option text to be 'Option 1', got '%s'", response.Options[0].Text.Text)
	}

	if response.Options[0].Value != "opt1" {
		t.Errorf("Expected first option value to be 'opt1', got '%s'", response.Options[0].Value)
	}
}

func TestHandleRequest_DirectJSON(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create a Slack request
	slackReq := SlackRequest{
		Type:     "block_suggestion",
		ActionID: "test_action",
		BlockID:  "test_block",
		Value:    "test",
	}

	// Convert to JSON
	jsonBody, err := json.Marshal(slackReq)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Add Slack signature headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := generateTestSignature(secret, timestamp, jsonBody)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := handleRequest(secret)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response SlackResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify options
	if len(response.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(response.Options))
	}
}

func TestHandleRequest_MissingPayload(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create form-encoded body WITHOUT payload field
	formData := url.Values{}
	formData.Set("other_field", "value")
	body := formData.Encode()

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add Slack signature headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := generateTestSignature(secret, timestamp, []byte(body))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := handleRequest(secret)
	handler.ServeHTTP(rr, req)

	// Check status code - should be 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleRequest_InvalidJSON(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create form-encoded body with invalid JSON in payload
	formData := url.Values{}
	formData.Set("payload", "{invalid json}")
	body := formData.Encode()

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add Slack signature headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := generateTestSignature(secret, timestamp, []byte(body))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := handleRequest(secret)
	handler.ServeHTTP(rr, req)

	// Check status code - should be 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
