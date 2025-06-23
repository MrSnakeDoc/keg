package deploy

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/MrSnakeDoc/keg/internal/install"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Deployer struct {
	Config *models.Config
	Runner runner.CommandRunner
}

func New(config *models.Config, r runner.CommandRunner) *Deployer {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Deployer{
		Config: config,
		Runner: r,
	}
}

func (d *Deployer) Execute() error {
	// 0. Check if Homebrew is installed
	isInstalled := utils.IsHomebrewInstalled()

	if isInstalled {
		logger.Success("Homebrew is already installed.")
		return nil
	}

	// 1. Check and install Homebrew
	if err := d.setupHomebrew(); err != nil {
		return err
	}

	// 2. Install brew plugins
	if err := d.ExecuteBrewPlugins(); err != nil {
		return err
	}

	logger.Success("Development environment deployed successfully!")
	return nil
}

func (d *Deployer) setupHomebrew() error {
	if _, err := exec.LookPath("zsh"); err != nil {
		return fmt.Errorf("please install ZSH first or restart your terminal with ZSH")
	}

	err := utils.SetHomebrewPath()
	if err != nil {
		return err
	}

	err = utils.ConfirmOrAbort("Homebrew is not installed. Do you want to install it? [y/N] ", "Homebrew installation canceled")
	if err != nil {
		return err
	}

	logger.Info("Installing Homebrew...")
	if err := d.installHomebrew(); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) installHomebrew() error {
	const brewInstallURL = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"
	installCmd := fmt.Sprintf(`/bin/bash -c "$(curl -fsSL %s)"`, brewInstallURL)

	// Show the user what will be executed
	logger.Warn("⚠️ Homebrew will be installed using the official script at:")
	logger.Warn("   %s", brewInstallURL)
	logger.Warn("🔗 You can audit it manually before continuing.")

	if err := utils.ConfirmOrAbort("Do you want to continue with the Homebrew installation? [y/N]", "Homebrew installation canceled"); err != nil {
		return err
	}

	if _, err := d.Runner.Run(context.Background(), 60*time.Second, runner.Stream, "bash", "-c", installCmd); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
	}

	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("homebrew installation succeeded but brew command not found in PATH")
	}

	return nil
}

func (d *Deployer) ExecuteBrewPlugins() error {
	logger.Info("Installing brew plugins...")

	inst := install.New(d.Config, d.Runner)
	if err := inst.Execute(nil, false); err != nil {
		return fmt.Errorf("failed to install brew plugins: %w", err)
	}

	return nil
}
