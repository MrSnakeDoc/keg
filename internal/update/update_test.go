package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type fakeHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f.DoFunc(req)
}

type mockChecker struct {
	resp *utils.VersionInfo
	err  error
}

func (m *mockChecker) Execute(ctx context.Context, checkOnly bool) (*utils.VersionInfo, error) {
	return m.resp, m.err
}

// Helper to create a fake binary and its sha256
func createFakeBinary(t *testing.T, content string) (string, string) {
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "keg")
	if err := os.WriteFile(binPath, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	return binPath, hex.EncodeToString(sum[:])
}

func TestUpdater_Execute_UpdateAvailable(t *testing.T) {
	origVersion := checker.Version
	checker.Version = "1.0.0"
	defer func() { checker.Version = origVersion }()

	ctx := context.Background()
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	const binContent = "FAKE_BINARY"
	_, sha := createFakeBinary(t, binContent)

	client := &fakeHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/repos/MrSnakeDoc/keg/releases/latest":
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewReader(
						[]byte(`{"tag_name":"v2.0.0"}`))),
				}, nil

			case "/MrSnakeDoc/keg/releases/download/v2.0.0/keg_2.0.0_linux_amd64":
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(binContent))),
				}, nil

			case "/MrSnakeDoc/keg/releases/download/v2.0.0/checksums.txt":
				line := sha + "  keg_2.0.0_linux_amd64\n"
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(line))),
				}, nil

			default:
				return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
			}
		},
	}

	cfg := config.DefaultUpdateConfig()
	updater := New(&cfg, client, nil)
	updater.pathInfo = &pathInfo{BinaryPath: filepath.Join(tmpHome, ".local", "bin", "keg")}

	if err := os.MkdirAll(filepath.Dir(updater.pathInfo.BinaryPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(updater.pathInfo.BinaryPath, []byte("OLD_BINARY"), 0o755); err != nil {
		t.Fatalf("write old bin: %v", err)
	}

	if err := updater.Execute(ctx, false); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	data, _ := os.ReadFile(updater.pathInfo.BinaryPath)
	if string(data) != binContent {
		t.Errorf("binary not updated, got: %s", data)
	}

	stateFile := filepath.Join(tmpHome, ".local", "state", "keg", "update-check.json")
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var st config.UpdateState
	if err := json.Unmarshal(stateData, &st); err != nil {
		t.Fatalf("failed to decode state file: %v", err)
	}

	if st.LatestVersion != "2.0.0" {
		t.Errorf("latest_version mismatch: got %s, want 2.0.0", st.LatestVersion)
	}
	if st.UpdateAvailable {
		t.Errorf("update_available should be false, got true")
	}
}

func TestUpdater_Execute_CheckOnly_UpdateAvailable(t *testing.T) {
	origVersion := checker.Version
	checker.Version = "1.0.0"
	defer func() { checker.Version = origVersion }()

	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	mockChecker := &mockChecker{resp: &utils.VersionInfo{Version: "2.0.0"}}
	up := New(nil, nil, mockChecker)

	err := up.Execute(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdater_Execute_CheckOnly_NoUpdate(t *testing.T) {
	origVersion := checker.Version
	checker.Version = "1.0.0"
	defer func() { checker.Version = origVersion }()

	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	mockChecker := &mockChecker{resp: nil}
	up := New(nil, nil, mockChecker)

	err := up.Execute(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdater_Execute_NoUpdateAvailable(t *testing.T) {
	origVersion := checker.Version
	checker.Version = "1.0.0"
	defer func() { checker.Version = origVersion }()

	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	mockChecker := &mockChecker{resp: nil}
	up := New(nil, nil, mockChecker)

	err := up.Execute(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdater_DownloadBinary_BadChecksum(t *testing.T) {
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	up := New(nil, nil, nil)
	up.response = &utils.VersionInfo{
		URL:    "http://fake",
		SHA256: "deadbeef",
	}
	up.pathInfo = &pathInfo{BinaryPath: filepath.Join(tmpHome, "keg")}

	up.Client = &fakeHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte("hello"))),
			}, nil
		},
	}

	err := up.downloadBinary(context.Background())
	if err == nil {
		t.Fatalf("expected checksum error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("expected checksum error, got %v", err)
	}
}

func TestUpdater_PrepareSwap_BrewPathError(t *testing.T) {
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	up := New(nil, nil, nil)
	up.pathInfo = &pathInfo{BinaryPath: filepath.Join(tmpHome, "doesnotexist")}

	utils.LookForFileInPath = func(_ string) (string, error) {
		return "/usr/local/bin/keg", nil
	}
	defer func() { utils.LookForFileInPath = utils.DefaultLookForFileInPath }()

	err := up.PrepareSwap()
	if err == nil {
		t.Fatalf("expected error for brew path, got nil")
	}
	if !strings.Contains(err.Error(), "/usr/local/bin") {
		t.Errorf("expected brew path error, got %v", err)
	}
}
