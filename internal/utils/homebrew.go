package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

// commandExists Verify if a command exists in the system
func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// IsHomebrewInstalled Verify if Homebrew is already installed
func IsHomebrewInstalled() bool {
	logger.Info("Checking Homebrew installation...")

	return CommandExists("brew")
}

func SetHomebrewPath() error {
	homebrewPath := "/home/linuxbrew/.linuxbrew"

	envVars := map[string]string{
		"HOMEBREW_PREFIX":     homebrewPath,
		"HOMEBREW_CELLAR":     homebrewPath + "/Cellar",
		"HOMEBREW_REPOSITORY": homebrewPath + "/Homebrew",
		"PATH":                homebrewPath + "/bin:" + os.Getenv("PATH"),
		"MANPATH":             homebrewPath + "/share/man:" + os.Getenv("MANPATH"),
		"INFOPATH":            homebrewPath + "/share/info:" + os.Getenv("INFOPATH"),
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	return nil
}

// InstalledSet returns a fast membership map of installed brew formulae.
// Key: package name, Value: true if installed.
func InstalledSet(r runner.CommandRunner) (map[string]bool, error) {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	// Force one package per line to simplify parsing
	out, err := r.Run(context.Background(), 60*time.Second, runner.Capture, "brew", "list", "--formula", "-1")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch installed packages: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	m := make(map[string]bool, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		m[name] = true
	}
	return m, nil
}

// ListInstalled returns the installed brew formulae as a slice (if you ever need it).
func ListInstalled(r runner.CommandRunner) ([]string, error) {
	set, err := InstalledSet(r)
	if err != nil {
		return nil, err
	}
	// Extract keys inline
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result, nil
}

// RunBrewCommand executes a brew command and handles warnings
func RunBrewCommand(r runner.CommandRunner, action, pkg string, ignoreWarnings []string) error {
	output, err := r.Run(context.Background(), 80*time.Second, runner.Capture, "brew", action, pkg)
	if err != nil {
		errStr := string(output)

		for _, warning := range ignoreWarnings {
			if strings.Contains(errStr, warning) {
				return nil
			}
		}

		return fmt.Errorf("brew %s failed for %s: %w", action, pkg, err)
	}

	return nil
}
