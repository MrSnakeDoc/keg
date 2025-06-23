package bootstraper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

func fakeBin(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("cannot create fake %s: %v", name, err)
	}
}

func expect(pm, mode string, pkgs []string) []string {
	switch pm {
	case "apt":
		if mode == "update" {
			return []string{"bash", "-c", "sudo apt update && sudo apt upgrade -y"}
		}
		return append([]string{"apt-get", "install", "-y"}, pkgs...)
	case "dnf":
		if mode == "update" {
			return []string{"dnf", "upgrade", "--refresh", "-y"}
		}
		return append([]string{"dnf", "install", "-y"}, pkgs...)
	case "pacman":
		if mode == "update" {
			return []string{"pacman", "-Syu", "--noconfirm"}
		}
		return append([]string{"pacman", "-S", "--noconfirm"}, pkgs...)
	}
	return nil
}

func TestRunPackageManagerCommand_AllManagers(t *testing.T) {
	cases := []struct {
		name string
		pm   string
		mode string
		cmd  packageManagerCommands
	}{
		{"apt-update", "apt", "update", packageManagerCommands{}},
		{"apt-install", "apt", "install", packageManagerCommands{install: []string{"zsh"}}},
		{"dnf-update", "dnf", "update", packageManagerCommands{}},
		{"dnf-install", "dnf", "install", packageManagerCommands{install: []string{"zsh"}}},
		{"pacman-update", "pacman", "update", packageManagerCommands{}},
		{"pacman-install", "pacman", "install", packageManagerCommands{install: []string{"zsh"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fakeBin(t, tmpDir, tc.pm)
			origPath := os.Getenv("PATH")
			defer utils.DeferRestore("PATH", origPath)
			utils.MustSet("PATH", tmpDir)

			// ── 2.  Bootstraper with MockRunner ───────────────────────────
			mockRun := runner.NewMockRunner()
			bs := New(&models.Config{}, mockRun)

			if err := bs.runPackageManagerCommand(tc.cmd); err != nil {
				t.Fatalf("runPackageManagerCommand err: %v", err)
			}

			want := expect(tc.pm, tc.mode, tc.cmd.install)
			if !mockRun.VerifyCommand("sudo", want...) {
				t.Fatalf("expected sudo %v, got %+v", want, mockRun.Commands)
			}
		})
	}
}
