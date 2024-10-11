package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	"fmt"
)

func TestLoadConfig(t *testing.T) {
	configContent := []byte(`
path_pattern: "test/*.json"
max_files: 5
api_endpoint: "http://localhost:8080/test"
api_token: "test_token"
server_key: "test_server_key"
check_interval: 10s
`)
	tempFile, err := ioutil.TempFile("", "config*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	loadConfig(tempFile.Name())

	if config.PathPattern != "test/*.json" {
		t.Errorf("Expected path_pattern to be 'test/*.json', got '%s'", config.PathPattern)
	}
	if config.MaxFiles != 5 {
		t.Errorf("Expected max_files to be 5, got %d", config.MaxFiles)
	}
	if config.ApiEndpoint != "http://localhost:8080/test" {
		t.Errorf("Expected api_endpoint to be 'http://localhost:8080/test', got '%s'", config.ApiEndpoint)
	}
	if config.ApiToken != "test_token" {
		t.Errorf("Expected api_token to be 'test_token', got '%s'", config.ApiToken)
	}
	if config.ServerKey != "test_server_key" {
		t.Errorf("Expected server_key to be 'test_server_key', got '%s'", config.ServerKey)
	}
	if config.CheckInterval != 10*time.Second {
		t.Errorf("Expected check_interval to be 10s, got '%s'", config.CheckInterval)
	}
}

func TestProcessFiles(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test_files")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test JSON files
	for i := 0; i < 3; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.json", i))
		content := []byte(`{"key": "value"}`)
		if err := ioutil.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}

	config.PathPattern = filepath.Join(tempDir, "*.json")
	config.MaxFiles = 5

	processFiles()
}

func TestSendPayload(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Expected Authorization 'Bearer test_token', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Server-Key") != "test_server_key" {
			t.Errorf("Expected X-Server-Key 'test_server_key', got '%s'", r.Header.Get("X-Server-Key"))
		}
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	config.ApiEndpoint = ts.URL
	config.ApiToken = "test_token"
	config.ServerKey = "test_server_key"

	payload := Payload{Data: []map[string]interface{}{{"key": "value"}}}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	success := sendPayload(payloadBytes)
	if !success {
		t.Errorf("Expected sendPayload to return true, got false")
	}
}
