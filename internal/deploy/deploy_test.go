package deploy

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
	dst := filepath.Join(dir, name)
	if err := os.WriteFile(dst, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("create fake %s: %v", name, err)
	}
}

func fakeStdin(t *testing.T, input string) (cleanup func()) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	utils.MustClose(w)

	orig := os.Stdin
	os.Stdin = r
	return func() { os.Stdin = orig }
}

func TestDeployer_Execute(t *testing.T) {
	tmp := t.TempDir()
	origPath := os.Getenv("PATH")
	defer utils.DeferRestore("PATH", origPath)

	t.Run("brew present → noop", func(t *testing.T) {
		fakeBin(t, tmp, "brew")
		utils.MustSet("PATH", tmp)

		d := New(&models.Config{}, runner.NewMockRunner())
		if err := d.Execute(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("install brew + plugins", func(t *testing.T) {
		tmp := t.TempDir()
		utils.MustSet("PATH", tmp)

		fakeBin(t, tmp, "zsh")
		mockRun := runner.NewMockRunner()
		d := New(&models.Config{}, mockRun)

		defer fakeStdin(t, "y\ny\ny\n")()

		mockRun.ResponseFunc = func(name string, _ ...string) ([]byte, error) {
			if name == "bash" {
				fakeBin(t, tmp, "brew")
			}
			return nil, nil
		}

		if err := d.Execute(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		installCmd := `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`

		if !mockRun.VerifyCommand("bash", "-c", installCmd) {
			t.Errorf("install script not executed; got %+v", mockRun.Commands)
		}
	})

	t.Run("user aborts", func(t *testing.T) {
		tmp := t.TempDir()
		utils.MustSet("PATH", tmp)

		fakeStdin(t, "n\n")()

		d := New(&models.Config{}, runner.NewMockRunner())
		if err := d.Execute(); err == nil {
			t.Fatalf("expected abort, got nil")
		}
	})
}
