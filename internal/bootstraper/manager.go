package bootstraper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

// Bootstraper manages the bootstrap process for setting up ZSH.
//
// Fields:
//   - Config: The configuration object containing user-defined settings.
//   - Runner: An instance of InteractiveRunner to execute commands.
//
// Description:
// This struct encapsulates the logic for installing and configuring ZSH as the default shell.
type Bootstraper struct {
	Runner runner.CommandRunner
}

// packageManagerCommands defines the commands for interacting with the system's package manager.
//
// Fields:
//   - install: A list of package names to install.
//
// Description:
// This struct is used to encapsulate package manager commands for installation or updates.
type packageManagerCommands struct {
	install []string
}

// New creates a new instance of Bootstraper.
//
// Parameters:
//   - config: The configuration object containing user-defined settings.
//   - r: An instance of InteractiveRunner to execute commands.
//
// Returns:
//   - *Bootstraper: A pointer to the newly created Bootstraper instance.
//
// Notes:
// If no runner is provided, a default StreamingRunner is used.
func New(r runner.CommandRunner) *Bootstraper {
	if r == nil {
		r = &runner.ExecRunner{}
	}
	return &Bootstraper{
		Runner: r,
	}
}

// Execute starts the bootstrap process for setting up ZSH.
//
// Returns:
//   - error: If any step in the bootstrap process fails.
//
// Behavior:
//   - Displays a warning message about the bootstrap process.
//   - Prompts the user for confirmation before proceeding.
//   - Calls setupZSH to handle the ZSH setup process.
//
// description:
// This method is the entry point for executing the bootstrap process.
// It handles user confirmation and calls the setupZSH method to perform the actual setup.
// It also handles any errors that may occur during the process.
func (d *Bootstraper) Execute() error {
	logger.Warn(`
	🛠️  Bootstrap: zsh installation and configuration
	- This command will help you set up zsh as your default shell.
	- This operation requires sudo privileges.
	- If you are not comfortable with the idea of granting sudo rights to KEG, please install zsh manually.
	- KEG will update your system package manager. (this requires sudo privileges).
	- KEG will check if zsh is installed.
	- If not installed, KEG will prompt you to install it (this requires sudo privileges).
	- KEG can set zsh as your default login shell (this also requires sudo privileges).
	- You will always be asked for confirmation before any change.
	- No actions will be taken without your explicit consent.
	`)

	if err := utils.ConfirmOrAbort("Do you want to continue? [y/N] ", "Bootstrap canceled by user"); err != nil {
		return err
	}

	if err := d.setupZSH(); err != nil {
		return err
	}
	return nil
}

func RunStream(ctx context.Context, r runner.CommandRunner,
	timeout time.Duration, name string, args ...string,
) error {
	_, err := r.Run(ctx, timeout, runner.Stream, name, args...)
	return err
}

// runPackageManagerCommand executes the appropriate package manager command.
//
// Parameters:
//   - commands: The packageManagerCommands struct containing the commands to execute.
//
// Returns:
//   - error: If the command execution fails.
//
// Behavior:
//   - Detects the system's package manager (apt, dnf, pacman).
//   - Executes the appropriate command for installation or updates.
func (b *Bootstraper) runPackageManagerCommand(cmds packageManagerCommands) error {
	pm, err := utils.PackageManager()
	if err != nil {
		return err
	}

	base := pm.Update
	if len(cmds.install) > 0 {
		base = append([]string{}, pm.Install...)
		base = append(base, cmds.install...)
	}

	return RunStream(context.Background(), b.Runner, 200*time.Second, "sudo", base...)
}

// updatePackageManagerIfNeeded prompts the user to update the system's package manager.
//
// Returns:
//   - error: If the update process fails or is canceled by the user.
//
// Behavior:
//   - Prompts the user for confirmation to update the package manager.
//   - Executes the update command if confirmed.
func (b *Bootstraper) updatePackageManagerIfNeeded() error {
	if err := utils.ConfirmOrAbort("Update system package manager? [y/N] ", "System package manager update canceled"); err != nil {
		return err
	}

	logger.Info("Updating system package manager...")

	if err := b.runPackageManagerCommand(packageManagerCommands{}); err != nil {
		return err
	}

	logger.Info("Checking ZSH installation...")
	return nil
}

// installZSH installs the ZSH shell using the system's package manager.
//
// Returns:
//   - error: If the installation process fails.
//
// Behavior:
//   - Executes the package manager command to install ZSH.
func (b *Bootstraper) installZSH() error {
	if err := b.runPackageManagerCommand(packageManagerCommands{install: []string{"zsh"}}); err != nil {
		return err
	}

	return nil
}

// checkAndInstallZSH checks if ZSH is installed and installs it if necessary.
//
// Returns:
//   - bool: True if ZSH was already installed, false otherwise.
//   - error: If the installation process fails.
//
// Behavior:
//   - Checks if ZSH is available on the system.
//   - Prompts the user for confirmation before installing ZSH.
func (d *Bootstraper) checkAndInstallZSH() (bool, error) {
	if _, err := exec.LookPath("zsh"); err == nil {
		return true, nil
	}

	if err := utils.ConfirmOrAbort("ZSH is not installed. Do you want to install it? [y/N] ", "ZSH installation canceled"); err != nil {
		return false, err
	}

	// Install ZSH
	if err := d.installZSH(); err != nil {
		return false, err
	}

	return false, nil
}

// changeDefaultShell sets ZSH as the default shell for the current user.
//
// Returns:
//   - error: If the shell change process fails.
//
// Behavior:
//   - Uses the `chsh` command to set ZSH as the default shell.
func (b *Bootstraper) changeDefaultShell() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to determine current user: %w", err)
	}
	username := currentUser.Username

	if _, err := b.Runner.Run(context.Background(), 60*time.Second, runner.Stream, "sudo", "chsh", "-s", "/bin/zsh", username); err != nil {
		return fmt.Errorf("failed to set ZSH as default shell: %w", err)
	}

	return nil
}

// setDefaultShell checks and sets ZSH as the default shell if necessary.
//
// Returns:
//   - bool: True if the shell was changed, false otherwise.
//   - error: If the shell change process fails.
//
// Behavior:
//   - Checks the current shell.
//   - Prompts the user for confirmation before changing the shell.
func (d *Bootstraper) setDefaultShell() (bool, error) {
	currentShell := os.Getenv("SHELL")
	if strings.Contains(currentShell, "zsh") {
		return false, nil
	}

	if err := utils.ConfirmOrAbort("ZSH is not set. Do you want to set it as the default shell? [y/N] ", "Shell change canceled"); err != nil {
		return false, err
	}

	if err := d.changeDefaultShell(); err != nil {
		return false, err
	}

	return true, nil
}

// showSetupMessages displays messages based on the ZSH setup process.
//
// Parameters:
//   - isInstalled: Indicates if ZSH was already installed.
//   - shellChanged: Indicates if the default shell was changed.
//
// Behavior:
//   - Displays success messages based on the setup outcome.
func (*Bootstraper) showSetupMessages(isInstalled, shellChanged bool) {
	if isInstalled && !shellChanged {
		logger.Success("ZSH is already installed and is the current shell.")
		return
	}

	if shellChanged {
		logger.Success("ZSH setup completed successfully.")
		logger.Info("Please restart your terminal to apply the changes.")
		logger.Info("After restarting your terminal you can run 'keg deploy' to complete the setup.")
		logger.Info("For more instructions, please refer to the README.")
	}
}

// setupZSH orchestrates the entire ZSH setup process.
//
// Returns:
//   - error: If any step in the setup process fails.
//
// Behavior:
//   - Updates the package manager if needed.
//   - Checks and installs ZSH if necessary.
//   - Sets ZSH as the default shell if required.
//   - Displays appropriate messages based on the setup outcome.
func (d *Bootstraper) setupZSH() error {
	if err := d.updatePackageManagerIfNeeded(); err != nil {
		return err
	}

	// Check if ZSH is already installed
	isInstalled, err := d.checkAndInstallZSH()
	if err != nil {
		return err
	}

	// Check and set default shell if needed
	shellChanged, err := d.setDefaultShell()
	if err != nil {
		return err
	}

	// Show appropriate messages based on what happened
	d.showSetupMessages(isInstalled, shellChanged)

	return nil
}
