package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/service"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type pathInfo struct {
	BinaryPath    string `json:"binary_path"`
	BackupPath    string `json:"backup_path,omitempty"`
	OldBinaryPath string `json:"old_binary_path,omitempty"`
	TempFileName  string `json:"temp_file_name,omitempty"`
}

type Updater struct {
	Config   *config.Config
	Client   service.HTTPClient
	Checker  checker.IChecker
	response *utils.VersionInfo
	pathInfo *pathInfo
}

func defaultBinaryPath() *pathInfo {
	return &pathInfo{
		BinaryPath: fmt.Sprintf("%s/.local/bin/keg", utils.GetHomeDir()),
	}
}

func New(conf *config.Config, client service.HTTPClient, chk checker.IChecker) *Updater {
	if conf == nil {
		def := config.DefaultUpdateConfig()
		conf = &def
	}

	if client == nil {
		client = service.NewHTTPClient(30 * time.Second)
	}

	if chk == nil {
		chk = checker.New(conf, client)
	}

	controller := &Updater{
		Config:   conf,
		Client:   client,
		Checker:  chk,
		response: &utils.VersionInfo{},
		pathInfo: defaultBinaryPath(),
	}

	return controller
}

func (u *Updater) checkUpdateState(ctx context.Context, checkOnly bool) (*utils.VersionInfo, error) {
	resp, err := u.Checker.Execute(ctx, checkOnly)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (u *Updater) Execute(ctx context.Context, checkOnly bool) error {
	logger.Info("🔄 Starting update process...")
	if checkOnly {
		logger.Info("🔄 Check-only mode: bypassing timer")
	}

	resp, err := u.checkUpdateState(ctx, checkOnly)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if checkOnly {
		if resp != nil {
			logger.Info("✅ Update available: v%s", resp.Version)
		} else {
			logger.Info("👍 Already on the latest version")
		}
		return nil
	}

	if resp == nil {
		logger.Info("No updates available")
		return nil
	}

	u.response = resp

	logger.Info("🔄 Update available: v%s", resp.Version)

	logger.Info("🔄 Starting downloading the binary file")
	err = u.downloadBinary(ctx)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	logger.Info("🔄 Preparing for binary swap...")
	err = u.PrepareSwap()
	if err != nil {
		return fmt.Errorf("failed to prepare swap: %w", err)
	}

	logger.Info("🔄 Starting binary swap...")
	err = u.ApplySwap()
	if err != nil {
		return fmt.Errorf("failed to apply swap: %w", err)
	}

	logger.Info("✅ New binary successfully moved to %s", u.pathInfo.BinaryPath)

	err = u.Cleanup()
	if err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	logger.Info("\n\nPlease run the following command to verify the installation:\n\n  keg --version\n")
	logger.Info("If you encounter any issues, please report them at: https://github.com/MrSnakeDoc/keg/issues\n")

	return nil
}

func (u *Updater) downloadBinary(ctx context.Context) error {
	dir := filepath.Dir(u.pathInfo.BinaryPath)
	file, err := os.CreateTemp(dir, "keg-update-*.bin")
	if err != nil {
		return err
	}
	u.pathInfo.TempFileName = file.Name()
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temp file %s: %w", file.Name(), err)
	}

	if err := service.DownloadToFile(ctx, u.Client, u.response.URL, u.pathInfo.TempFileName, 0); err != nil {
		return err
	}

	return utils.ValidateSHA256Checksum(u.pathInfo.TempFileName, u.response.SHA256)
}

func (u *Updater) PrepareSwap() error {
	if ok, _ := utils.FileExists(u.pathInfo.BinaryPath); !ok {
		logger.Warn("🔍 Target path %s not found, trying to locate it...", u.pathInfo.BinaryPath)

		expandedPath, err := utils.LookForFileInPath("keg")
		if err != nil {
			return fmt.Errorf("unable to locate existing keg binary: %w", err)
		}
		u.pathInfo.OldBinaryPath = strings.TrimSpace(expandedPath)
		logger.Info("🔎 Found existing keg binary at %s", u.pathInfo.OldBinaryPath)

		if strings.HasPrefix(u.pathInfo.OldBinaryPath, "linuxbrew/.linuxbrew") {
			utils.WarnBrewInstallation(".linuxbrew")
			return fmt.Errorf("keg binary found in linuxbrew/.linuxbrew, please remove it before proceeding")
		}

		if strings.HasPrefix(u.pathInfo.OldBinaryPath, "/usr/local/bin") {
			utils.WarnBrewInstallation("/usr/local/bin")
			return fmt.Errorf("keg binary found in /usr/local/bin, please remove it before proceeding")
		}
	}

	u.pathInfo.BackupPath = u.pathInfo.BinaryPath + ".old"

	logger.Info("🔄 Backing up current executable to %s", u.pathInfo.BackupPath)

	return nil
}

func (u *Updater) ApplySwap() error {
	// 1. Rename existing binary -> .old  (atomic)
	if err := os.Rename(u.pathInfo.BinaryPath, u.pathInfo.BackupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	// Defer rollback in case there is a crash on next step
	defer func() {
		if rec := recover(); rec != nil {
			_ = os.Rename(u.pathInfo.BackupPath, u.pathInfo.BinaryPath)
			panic(rec)
		}
	}()

	// 2. Rename new file -> final binary (atomic)
	if err := os.Rename(u.pathInfo.TempFileName, u.pathInfo.BinaryPath); err != nil {
		// rollback
		_ = os.Rename(u.pathInfo.BackupPath, u.pathInfo.BinaryPath)
		return fmt.Errorf("install failed: %w", err)
	}
	// 3. Chmod after rename to ensure permissions are set correctly
	return os.Chmod(u.pathInfo.BinaryPath, 0o755)
}

func (u *Updater) Cleanup() error {
	home := utils.GetHomeDir()
	stateFile := filepath.Join(home, ".local", "state", "keg", "update-check.json")
	logger.Info("🔄 Updating state file at %s...", stateFile)

	state := config.UpdateState{
		LastChecked:     time.Now().UTC(),
		LatestVersion:   u.response.Version,
		UpdateAvailable: false,
	}

	if err := utils.CreateFile(stateFile, state, utils.FileTypeJSON, 0o644); err != nil {
		return fmt.Errorf("failed to update state file: %w", err)
	}

	return nil
}
