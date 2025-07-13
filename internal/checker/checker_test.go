package checker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

const (
	removeFile       = true
	withExistingFile = true
	cleanup          = false
)

func TestCheckerController_Execute(t *testing.T) {
	// Create two servers: one for release info, one for checksums
	releaseServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		mockRelease := GitHubRelease{
			TagName:     "v1.2.3",
			Name:        "Release v1.2.3",
			Draft:       false,
			Prerelease:  false,
			PublishedAt: "2025-05-31T15:30:00Z",
		}
		if err := json.NewEncoder(w).Encode(mockRelease); err != nil {
			t.Fatalf("Failed to encode mock release: %v", err)
		}
	}))
	defer releaseServer.Close()

	checksumServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Mock checksums.txt content
		checksums := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  keg_1.2.3_linux_amd64\n"
		if _, err := w.Write([]byte(checksums)); err != nil {
			logger.LogError("Failed to write mock checksums: %v", err)
		}
	}))
	defer checksumServer.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home directory: %v", err)
	}
	stateFile := filepath.Join(home, ".local", "state", "keg", "update-check.json")

	// Cleanup the state file if it exists
	if removeFile == true {
		_ = os.Remove(stateFile)
	}

	// Create the state file with correct format
	if withExistingFile == true {
		initialState := config.UpdateState{
			LastChecked:     time.Date(2025, 5, 15, 11, 19, 51, 0, time.UTC),
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		}
		if err := utils.CreateFile(stateFile, initialState, "json", 0o644); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}
	}

	conf := &config.Config{
		VersionURL:      releaseServer.URL,
		ChecksumBaseURL: checksumServer.URL,
		CheckFrequency:  1 * time.Millisecond,
	}

	Version = "1.0.0"
	checkerController := New(context.Background(), conf, releaseServer.Client())

	_, err = checkerController.Execute(false)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if exists, _ := utils.FileExists(stateFile); !exists {
		t.Fatalf("State file not created: %s", stateFile)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var state config.UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Failed to unmarshal state file: %v", err)
	}

	if state.LatestVersion != "1.2.3" {
		t.Errorf("Expected version 1.2.3, got %s", state.LatestVersion)
	}

	if !state.UpdateAvailable {
		t.Errorf("Expected update to be available, got false")
	}

	// cleanup the state file
	if cleanup == true {
		_ = os.Remove(stateFile)
	}
}
