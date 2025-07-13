package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/service"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type CheckerController struct {
	Config     config.Config
	HTTPClient service.HTTPClient
	ctx        context.Context
	cancel     context.CancelFunc
	response   *utils.VersionInfo
}

type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
}

func New(ctx context.Context, conf *config.Config, client service.HTTPClient) *CheckerController {
	if conf == nil {
		defaultConfig := config.DefaultCheckerConfig()
		conf = &defaultConfig
	}

	if client == nil {
		client = service.NewHTTPClient(30 * time.Second)
	}

	controller := &CheckerController{
		Config:     *conf,
		HTTPClient: client,
		ctx:        ctx,
		cancel:     func() {},
		response:   &utils.VersionInfo{},
	}

	return controller
}

func (c *CheckerController) Execute(checkOnly bool) (*utils.VersionInfo, error) {
	state, err := loadUpdateState()
	if err != nil {
		logger.Debug("Failed to load update state: %v", err)
	}

	needsCheck := checkOnly || state == nil || time.Since(state.LastChecked) >= c.Config.CheckFrequency

	if needsCheck {
		var resp *utils.VersionInfo
		resp, err = c.checkUpdate()
		if err != nil {
			logger.Debug("Failed to check for updates: %v", err)
			return nil, nil
		}

		if c.Config.ForceBypassSave {
			return resp, nil
		}
	}

	return nil, nil
}

func loadUpdateState() (*config.UpdateState, error) {
	var state config.UpdateState

	updateStateFile, err := utils.EnsureUpdateStateFileExists()
	if err != nil {
		logger.Debug("Failed to ensure update state file exists: %v", err)
		return nil, fmt.Errorf("failed to ensure update state file exists: %w", err)
	}

	if err := utils.FileReader(updateStateFile, "json", &state); err != nil {
		logger.Debug("failed to read update state: %w", err)
		return nil, fmt.Errorf("failed to read update state: %w", err)
	}

	return &state, nil
}

func saveUpdateState(state config.UpdateState) error {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Debug("Failed to get user home directory: %v", err)
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	stateFile := filepath.Join(home, ".local", "state", "keg", "update-check.json")

	if err := utils.CreateFile(stateFile, state, "json", 0o644); err != nil {
		logger.Debug("Failed to create update state file: %v", err)
		return fmt.Errorf("failed to create update state file: %w", err)
	}

	return nil
}

func MakeHTTPRequest(ctx context.Context, client service.HTTPClient, url string) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		logger.Debug("Context error: %v", err)
		return nil, err
	}

	parsedURL, err := utils.ParseSecureURL(url)
	if err != nil {
		logger.Debug("Failed to parse URL: %v", err)
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), http.NoBody)
	if err != nil {
		logger.Debug("Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("Failed to perform request: %v", err)
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Debug("Received non-200 response: %d", resp.StatusCode)
		return nil, fmt.Errorf("non-200 response: %d", resp.StatusCode)
	}

	return resp, nil
}

func convertReleaseToVersionInfo(release *GitHubRelease, checksum string) *utils.VersionInfo {
	version := strings.TrimPrefix(release.TagName, "v")

	// Build download URL (adjust based on your actual release assets)
	name := utils.AssetName(version)
	downloadURL := fmt.Sprintf(
		"https://github.com/MrSnakeDoc/keg/releases/download/%s/%s",
		release.TagName, name)

	return &utils.VersionInfo{
		Version: version,
		URL:     downloadURL,
		SHA256:  checksum,
	}
}

func (c *CheckerController) fetchChecksum(release *GitHubRelease) (string, error) {
	baseURL := c.Config.ChecksumBaseURL
	if baseURL == "" {
		baseURL = "https://github.com/MrSnakeDoc/keg/releases/download"
	}
	// Build checksums URL
	checksumsURL := fmt.Sprintf("%s/%s/checksums.txt", baseURL, release.TagName)

	resp, err := MakeHTTPRequest(c.ctx, c.HTTPClient, checksumsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksums: %w", err)
	}

	// Read the entire checksums file
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse checksums to find our binary
	return utils.ParseChecksumsForBinary(string(body), release.TagName)
}

func (c *CheckerController) checkUpdate() (*utils.VersionInfo, error) {
	resp, err := MakeHTTPRequest(c.ctx, c.HTTPClient, c.Config.VersionURL)
	if err != nil {
		logger.Debug("Failed to make HTTP request: %v", err)
		return nil, nil
	}
	defer utils.Try(resp.Body.Close)

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logger.Debug("Failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Second call: get checksums
	checksum, err := c.fetchChecksum(&release)
	if err != nil {
		logger.Debug("Failed to fetch checksum: %v", err)
		checksum = "" // Fallback to empty checksum if fetching fails
	}

	// Convert GitHub release to VersionInfo
	versionInfo := convertReleaseToVersionInfo(&release, checksum)

	if err := utils.ValidateVersion(versionInfo); err != nil {
		logger.Debug("Version validation failed: %v", err)
		return nil, fmt.Errorf("version validation failed: %w", err)
	}

	// Store the response
	c.response = versionInfo

	isNewer, err := c.isUpdateAvailable(versionInfo)
	if err != nil {
		logger.Debug("Failed to check if update is available: %v", err)
		return nil, fmt.Errorf("failed to check if update is available: %w", err)
	}

	if !c.Config.ForceBypassSave {
		if err := c.updateState(isNewer, versionInfo); err != nil {
			logger.Debug("Failed to update state: %v", err)
			return nil, fmt.Errorf("failed to update state: %w", err)
		}
	}

	responses := []*utils.VersionInfo{nil, c.response}
	return responses[utils.BoolToInt(isNewer)], nil
}

func (c *CheckerController) isUpdateAvailable(info *utils.VersionInfo) (bool, error) {
	isNewer, err := utils.IsNewerVersion(info.Version, Version)
	if err != nil {
		logger.Debug("Failed to compare versions: %v", err)
		return false, fmt.Errorf("failed to compare versions: %w", err)
	}
	return isNewer, nil
}

func (c *CheckerController) updateState(isNewer bool, info *utils.VersionInfo) error {
	now := time.Now().UTC()

	baseState := config.UpdateState{LastChecked: now}

	version := map[bool]string{
		true:  info.Version,
		false: Version,
	}[isNewer]

	state := utils.NewUpdateState(baseState, isNewer, version)

	if err := saveUpdateState(state); err != nil {
		logger.Debug("Failed to save update state: %v", err)
		return fmt.Errorf("failed to save update state: %w", err)
	}

	return nil
}
