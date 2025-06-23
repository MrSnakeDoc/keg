package notifier

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"No ANSI", "Hello World", "Hello World"},
		{"With Color", "\033[31mRed\033[0m", "Red"},
		{"Multiple Colors", "\033[32mGreen\033[0m \033[34mBlue\033[0m", "Green Blue"},
		{"Complex ANSI", "\033[1;38;5;39mAzure Blue\033[0m", "Azure Blue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetMaxWidth(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected int
	}{
		{"Empty", []string{}, 0},
		{"Single Line", []string{"Hello"}, 5},
		{"Multiple Lines", []string{"Hello", "World", "Testing"}, 7},
		{"With ANSI", []string{"\033[31mRed\033[0m", "\033[32mGreen\033[0m"}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GetMaxWidth(tt.lines)
			if result != tt.expected {
				t.Errorf("getMaxWidth(%v) = %d, want %d", tt.lines, result, tt.expected)
			}
		})
	}
}

func TestDisplayVersionUpdate(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayVersionUpdate("1.2.3")

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe: %v", err)
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "New Version Available!") {
		t.Errorf("Output should contain 'New Version Available!': %s", output)
	}
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("Output should contain version '1.2.3': %s", output)
	}
	if !strings.Contains(output, "keg update") {
		t.Errorf("Output should contain 'keg update' command: %s", output)
	}
}

func fileHandling(t *testing.T) string {
	tempDir := filepath.Join(os.TempDir(), "keg-test-"+time.Now().Format("20060102150405"))
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal("⚠️ Failed to remove temporary directory")
		}
	}()
	return tempDir
}

func handleHomeDirectory(t *testing.T, tempDir string) string {
	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	return originalHome
}

func handleFullPath(t *testing.T, tempDir string) string {
	stateDir := filepath.Join(tempDir, ".local", "state", "keg")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("Failed to create state directory: %v", err)
	}

	stateFile := filepath.Join(stateDir, "update-check.json")

	return stateFile
}

func deleteFileIfExists(t *testing.T, stateFile string) {
	if ok, _ := utils.FileExists(stateFile); ok {
		if err := os.Remove(stateFile); err != nil {
			t.Fatalf("Failed to remove state file: %v", err)
		}
	}
}

func TestDisplayUpdateNotification(t *testing.T) {
	tempDir := fileHandling(t)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	originalHome := handleHomeDirectory(t, tempDir)

	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Fatal("⚠️ Failed to remove temporary directory")
		}
	}()

	stateFile := handleFullPath(t, tempDir)

	t.Run("No state file", func(t *testing.T) {
		deleteFileIfExists(t, stateFile)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		DisplayUpdateNotification()

		if err := w.Close(); err != nil {
			t.Fatalf("Failed to close pipe: %v", err)
		}
		os.Stdout = oldStdout

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}
		output := buf.String()

		if !strings.Contains(output, "Update state file does not exist") {
			t.Errorf("Expected error message about missing state file, got: %s", output)
		}
	})

	t.Run("Update available", func(t *testing.T) {
		state := config.UpdateState{
			LastChecked:     time.Now(),
			UpdateAvailable: true,
			LatestVersion:   "2.0.0",
		}

		if err := utils.CreateFile(stateFile, state, "json", 0o644); err != nil {
			t.Fatalf("Failed to create state file: %v", err)
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		DisplayUpdateNotification()

		if err := w.Close(); err != nil {
			t.Fatalf("Failed to close pipe: %v", err)
		}
		os.Stdout = oldStdout

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}
		output := buf.String()

		if !strings.Contains(output, "New Version Available!") {
			t.Errorf("Expected update notification, got: %s", output)
		}
		if !strings.Contains(output, "2.0.0") {
			t.Errorf("Expected version 2.0.0 in notification, got: %s", output)
		}
	})

	t.Run("No update available", func(t *testing.T) {
		state := config.UpdateState{
			LastChecked:     time.Now(),
			UpdateAvailable: false,
			LatestVersion:   checker.Version,
		}

		if err := utils.CreateFile(stateFile, state, "json", 0o644); err != nil {
			t.Fatalf("Failed to create state file: %v", err)
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		DisplayUpdateNotification()

		if err := w.Close(); err != nil {
			t.Fatalf("Failed to close pipe: %v", err)
		}
		os.Stdout = oldStdout

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}
		output := buf.String()

		if output != "" {
			t.Errorf("Expected no output, got: %s", output)
		}
	})
}
