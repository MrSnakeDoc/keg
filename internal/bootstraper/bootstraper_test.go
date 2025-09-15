package bootstraper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func sawChshToZsh(m *runner.MockRunner) bool {
	for _, c := range m.Commands {
		if c.Name == "sudo" && len(c.Args) >= 4 &&
			c.Args[0] == "chsh" &&
			c.Args[1] == "-s" &&
			c.Args[2] == "/bin/zsh" {
			return true // ignore the actual username
		}
	}
	return false
}

func expect(pm, mode string, pkgs []string) []string {
	switch pm {
	case "apt":
		if mode == "update" {
			return []string{"bash", "-c", "sudo apt update && sudo apt upgrade -y"}
		}
		return append([]string{"apt", "install", "-y"}, pkgs...)
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
			bs := New(mockRun)

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

func TestCheckAndInstallZSH_AlreadyInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	fakeBin(t, os.Getenv("PATH"), "zsh")

	mr := runner.NewMockRunner()
	bs := New(mr)

	got, err := bs.checkAndInstallZSH()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got {
		t.Fatalf("expected already-installed=true")
	}
	if len(mr.Commands) != 0 {
		t.Fatalf("expected no sudo calls, got: %+v", mr.Commands)
	}
}

func TestSetDefaultShell_ShouldChange(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	restore := ConfirmOrAbortFn
	ConfirmOrAbortFn = func(string, string) error { return nil } // user accepts
	defer func() { ConfirmOrAbortFn = restore }()

	mr := runner.NewMockRunner()
	bs := New(mr)

	changed, err := bs.setDefaultShell()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatalf("expected shellChanged=true")
	}
	if !sawChshToZsh(mr) {
		t.Fatalf("expected sudo chsh -s /bin/zsh <user>, got: %+v", mr.Commands)
	}
}

func TestUpdatePM_Refused(t *testing.T) {
	restore := ConfirmOrAbortFn
	ConfirmOrAbortFn = func(_, _ string) error { return fmt.Errorf("canceled") }
	defer func() { ConfirmOrAbortFn = restore }()

	// fake pacman in PATH to avoid PM detection error
	tmp := t.TempDir()
	fakeBin(t, tmp, "pacman")
	t.Setenv("PATH", tmp)

	mr := runner.NewMockRunner()
	bs := New(mr)
	if err := bs.updatePackageManagerIfNeeded(); err == nil {
		t.Fatalf("expected user-canceled error")
	}
	if len(mr.Commands) != 0 {
		t.Fatalf("expected no sudo/pm calls")
	}
}

func TestExecute_Cancelled(t *testing.T) {
	restore := ConfirmOrAbortFn
	ConfirmOrAbortFn = func(_, msg string) error { return errors.New(msg) }
	defer func() { ConfirmOrAbortFn = restore }()

	mr := runner.NewMockRunner()
	bs := New(mr)

	if err := bs.Execute(); err == nil || err.Error() != "Bootstrap canceled by user" {
		t.Fatalf("expected cancel error, got %v", err)
	}
}

func TestInstallZSH(t *testing.T) {
	tmp := t.TempDir()
	fakeBin(t, tmp, "apt")
	t.Setenv("PATH", tmp)

	mr := runner.NewMockRunner()
	bs := New(mr)

	if err := bs.installZSH(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if !mr.VerifyCommand("sudo", "apt", "install", "-y", "zsh") {
		t.Fatalf("expected sudo apt install -y zsh, got %+v", mr.Commands)
	}
}

func TestChangeDefaultShell(t *testing.T) {
	mr := runner.NewMockRunner()
	bs := New(mr)

	if err := bs.changeDefaultShell(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if !sawChshToZsh(mr) {
		t.Fatalf("expected sudo chsh -s /bin/zsh <user>, got %+v", mr.Commands)
	}
}

func TestShowSetupMessages(t *testing.T) {
	bs := New(runner.NewMockRunner())

	bs.showSetupMessages(true, false)

	bs.showSetupMessages(false, true)
}

func TestUpdatePM_Success(t *testing.T) {
	restore := ConfirmOrAbortFn
	ConfirmOrAbortFn = func(_, _ string) error { return nil } // user accepts
	defer func() { ConfirmOrAbortFn = restore }()

	tmp := t.TempDir()
	fakeBin(t, tmp, "apt")
	t.Setenv("PATH", tmp)

	mr := runner.NewMockRunner()
	bs := New(mr)

	if err := bs.updatePackageManagerIfNeeded(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(mr.Commands) == 0 {
		t.Fatalf("expected package manager update command, got none")
	}
}

func TestSetupZSH_AllCases(t *testing.T) {
	restore := ConfirmOrAbortFn
	defer func() { ConfirmOrAbortFn = restore }()

	// 1. Fake package manager so utils.PackageManager() succeeds
	tmp := t.TempDir()
	fakeBin(t, tmp, "apt")
	t.Setenv("PATH", tmp)

	// 2. Fake shell so setDefaultShell() triggers
	t.Setenv("SHELL", "/bin/bash")

	// 3. User always says yes
	ConfirmOrAbortFn = func(_, _ string) error { return nil }

	// 4. Mock runner simulates chsh failing
	mr := runner.NewMockRunner()
	mr.ResponseFunc = func(name string, args ...string) ([]byte, error) {
		if name == "sudo" && len(args) > 0 && args[0] == "chsh" {
			return nil, fmt.Errorf("fail chsh")
		}
		return nil, nil
	}

	bs := New(mr)

	// 5. Execute setupZSH and expect error from chsh
	if err := bs.setupZSH(); err == nil || !strings.Contains(err.Error(), "fail chsh") {
		t.Fatalf("expected fail chsh error, got %v", err)
	}
}
