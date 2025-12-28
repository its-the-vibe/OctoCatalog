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

// setupTestCatalogWithMoreOptions initializes a test catalog with more options for filtering tests
func setupTestCatalogWithMoreOptions() {
	catalog = []CatalogEntry{
		{
			ActionID: "test_action",
			Options: []Option{
				{Text: "InnerGate", Value: "InnerGate"},
				{Text: "OctoSlack", Value: "OctoSlack"},
				{Text: "Poppit", Value: "Poppit"},
				{Text: "SlackLiner", Value: "SlackLiner"},
				{Text: "Gateway", Value: "Gateway"},
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
		Value:    "",
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
		Value:    "",
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

func TestHandleRequest_UnsupportedContentType(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create request with unsupported content type
	body := []byte("some data")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "text/plain")

	// Add Slack signature headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := generateTestSignature(secret, timestamp, body)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := handleRequest(secret)
	handler.ServeHTTP(rr, req)

	// Check status code - should be 415 Unsupported Media Type
	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
	}
}

func TestHandleRequest_FormEncodedWithCharset(t *testing.T) {
	setupTestCatalog()
	secret := "test-secret"

	// Create a Slack request
	slackReq := SlackRequest{
		Type:     "block_suggestion",
		ActionID: "test_action",
		BlockID:  "test_block",
		Value:    "",
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

	// Create test request with charset in content type
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

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
}

func TestHandleRequest_FilterByValue_EmptyQuery(t *testing.T) {
setupTestCatalogWithMoreOptions()
secret := "test-secret"

// Create a Slack request with empty value (should return all options)
slackReq := SlackRequest{
Type:     "block_suggestion",
ActionID: "test_action",
BlockID:  "test_block",
Value:    "",
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

// Verify all 5 options are returned when query is empty
if len(response.Options) != 5 {
t.Errorf("Expected 5 options, got %d", len(response.Options))
}
}

func TestHandleRequest_FilterByValue_MatchingText(t *testing.T) {
setupTestCatalogWithMoreOptions()
secret := "test-secret"

// Create a Slack request with a query that matches some options by text
slackReq := SlackRequest{
Type:     "block_suggestion",
ActionID: "test_action",
BlockID:  "test_block",
Value:    "Slack",
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

// Should match "OctoSlack" and "SlackLiner" (2 options)
if len(response.Options) != 2 {
t.Errorf("Expected 2 options, got %d", len(response.Options))
}

// Verify the matched options
foundOctoSlack := false
foundSlackLiner := false
for _, opt := range response.Options {
if opt.Text.Text == "OctoSlack" {
foundOctoSlack = true
}
if opt.Text.Text == "SlackLiner" {
foundSlackLiner = true
}
}

if !foundOctoSlack {
t.Error("Expected to find 'OctoSlack' in results")
}
if !foundSlackLiner {
t.Error("Expected to find 'SlackLiner' in results")
}
}

func TestHandleRequest_FilterByValue_CaseInsensitive(t *testing.T) {
setupTestCatalogWithMoreOptions()
secret := "test-secret"

// Create a Slack request with a lowercase query
slackReq := SlackRequest{
Type:     "block_suggestion",
ActionID: "test_action",
BlockID:  "test_block",
Value:    "gate",
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

// Should match "InnerGate" and "Gateway" (case-insensitive)
if len(response.Options) != 2 {
t.Errorf("Expected 2 options, got %d", len(response.Options))
}

// Verify the matched options
foundInnerGate := false
foundGateway := false
for _, opt := range response.Options {
if opt.Text.Text == "InnerGate" {
foundInnerGate = true
}
if opt.Text.Text == "Gateway" {
foundGateway = true
}
}

if !foundInnerGate {
t.Error("Expected to find 'InnerGate' in results")
}
if !foundGateway {
t.Error("Expected to find 'Gateway' in results")
}
}

func TestHandleRequest_FilterByValue_NoMatch(t *testing.T) {
setupTestCatalogWithMoreOptions()
secret := "test-secret"

// Create a Slack request with a query that doesn't match anything
slackReq := SlackRequest{
Type:     "block_suggestion",
ActionID: "test_action",
BlockID:  "test_block",
Value:    "xyz123",
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

// Should return empty list when nothing matches
if len(response.Options) != 0 {
t.Errorf("Expected 0 options, got %d", len(response.Options))
}
}

func TestHandleRequest_FilterByValue_MatchByValue(t *testing.T) {
setupTestCatalogWithMoreOptions()
secret := "test-secret"

// Create a Slack request with a query that matches by value field
slackReq := SlackRequest{
Type:     "block_suggestion",
ActionID: "test_action",
BlockID:  "test_block",
Value:    "Poppit",
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

// Should match "Poppit" by value
if len(response.Options) != 1 {
t.Errorf("Expected 1 option, got %d", len(response.Options))
}

if len(response.Options) > 0 && response.Options[0].Text.Text != "Poppit" {
t.Errorf("Expected 'Poppit', got '%s'", response.Options[0].Text.Text)
}
}
