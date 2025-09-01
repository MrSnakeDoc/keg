package core

import (
	"errors"
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/MrSnakeDoc/keg/internal/versions"
)

var ErrPkgNotFound = errors.New("package not found in configuration")

var pastTense = map[string]string{
	"install":   "installed",
	"upgrade":   "upgraded",
	"uninstall": "uninstalled",
}

// Controller defines the behavior required for managing software packages.
//
// Implementations of this interface must be able to:
//   - Check if a package is already installed
//   - Locate a package based on a provided name
//   - Retrieve the display name for a package (binary or command)
type Controller interface {
	IsPackageInstalled(name string) bool
	FindPackage(name string) *models.Package
	GetPackageName(pkg *models.Package) string
}

// PackageAction describes the operation to perform on a package.
//
// Fields:
//   - Name: The name of the package to operate on
//   - ActionVerb: The action to execute (e.g. "install", "upgrade", "uninstall")
//   - SkipMessage: Optional message shown when skipping a package
//
// Description:
// This struct encapsulates the action to be performed on a package,
type PackageAction struct {
	Name        string
	ActionVerb  string
	SkipMessage string
}

// Base is the core implementation of the Controller interface.
//
// Fields:
//   - Config: The user configuration containing package definitions
//   - installedPkgs: A cache of installed packages to avoid repeated checks
//   - Runner: A CommandRunner instance to execute system commands
//
// It stores the user configuration, the internal cache of installed packages,
// and uses a CommandRunner to interact with the underlying system.
type Base struct {
	Config        *models.Config
	installedPkgs map[string]bool
	Runner        runner.CommandRunner
}

// PackageHandlerOptions defines the behavior of how packages should be processed.
//
// Fields:
//   - Action: The PackageAction to apply
//   - Packages: List of package names to target (optional)
//   - FilterFunc: Function to include/exclude packages from bulk operations
//   - ValidateFunc: Function to validate a package name before acting on it
//
// Description:
// This struct allows for flexible handling of packages, including filtering
type PackageHandlerOptions struct {
	Action       PackageAction
	Packages     []string
	FilterFunc   func(*models.Package) bool
	ValidateFunc func(string) bool
	AllowAdHoc   bool
}

// NewBase instantiates a new Base struct.
//
// Parameters:
//   - config: the main configuration struct loaded by the user
//   - r: command runner abstraction for executing system commands
//
// Returns:
//   - *Base: pointer to the new Base instance with initialized state
func NewBase(config *models.Config, r runner.CommandRunner) *Base {
	return &Base{
		Config:        config,
		installedPkgs: make(map[string]bool),
		Runner:        r,
	}
}

// FindPackage attempts to locate a package from the configuration based on its name.
//
// Parameters:
//   - name: the name to search for (matches .Command or .Binary fields)
//
// Returns:
//   - *models.Package: pointer to the matched package
//   - bool: true if the package was found, false otherwise
func (b *Base) FindPackage(name string) (*models.Package, bool) {
	for idx := range b.Config.Packages {
		pkg := &b.Config.Packages[idx]
		if pkg.Command == name || (pkg.Binary != "" && pkg.Binary == name) {
			return pkg, true
		}
	}
	return nil, false
}

// IsPackageInstalled determines whether the given package is currently installed.
//
// Parameters:
//   - name: the name of the package to check
//
// Returns:
//   - bool: true if the package is installed, false otherwise
//
// Notes:
//   - This function lazily loads the installed package list once on first call.
func (b *Base) IsPackageInstalled(name string) bool {
	if len(b.installedPkgs) == 0 {
		if err := b.loadInstalledPackages(); err != nil {
			return false
		}
	}
	return b.installedPkgs[name]
}

// loadInstalledPackages fetches the list of installed packages from the system
// using the provided runner and a mapping function.
//
// Returns:
//   - error: non-nil if the package map could not be loaded
func (b *Base) loadInstalledPackages() error {
	m, err := utils.MapInstalledPackagesWith(b.Runner, func(pkg string) (string, bool) {
		return pkg, true
	})
	if err != nil {
		return err
	}
	b.installedPkgs = m
	return nil
}

// GetPackageName returns the most appropriate name to use for a package.
//
// Parameters:
//   - pkg: the package whose name is to be retrieved
//
// Returns:
//   - string: pkg.Binary if available, otherwise pkg.Command
func (*Base) GetPackageName(pkg *models.Package) string {
	if pkg.Binary != "" {
		return pkg.Binary
	}
	return pkg.Command
}

// DefaultPackageHandlerOptions returns a default configuration for handling packages.
//
// Parameters:
//   - action: the action to be performed (install, upgrade, etc.)
//
// Returns:
//   - PackageHandlerOptions: pre-filled options with sensible defaults
//
// Notes:
//   - Skips optional packages
//   - Accepts all input as valid (no strict validation)
func DefaultPackageHandlerOptions(action PackageAction) PackageHandlerOptions {
	return PackageHandlerOptions{
		Action:       action,
		FilterFunc:   func(p *models.Package) bool { return !p.Optional },
		ValidateFunc: func(string) bool { return true },
		AllowAdHoc:   false,
	}
}

// HandlePackages performs the given action on a filtered list of packages.
//
// Parameters:
//   - opts: options that define which packages to act on and how
//
// Returns:
//   - error: if one or more packages fail to process
//
// Behavior:
//   - If opts.Packages is non-empty, only those packages are handled
//   - Otherwise, all packages passing FilterFunc are considered
func (b *Base) HandlePackages(opts PackageHandlerOptions) error {
	if len(opts.Packages) > 0 {
		for _, pkgName := range opts.Packages {
			if err := b.handleSelectedPackage(opts.Action, pkgName, opts.ValidateFunc, opts.AllowAdHoc); err != nil {
				return fmt.Errorf("failed to %s package %s: %w", opts.Action.ActionVerb, pkgName, err)
			}
		}
		return nil
	}

	for _, pkg := range b.Config.Packages {
		if !opts.FilterFunc(&pkg) {
			continue
		}
		name := b.GetPackageName(&pkg)
		if err := b.handleSelectedPackage(opts.Action, name, opts.ValidateFunc, opts.AllowAdHoc); err != nil {
			return fmt.Errorf("failed to %s package %s: %w", opts.Action.ActionVerb, name, err)
		}
	}
	return nil
}

// resolvePackageScoped behaves like resolvePackage, but if not found in the
// manifest and allowAdHoc is true, it will accept locally installed packages
// by synthesizing a minimal package definition (Command = name).
func (b *Base) resolvePackageScoped(name string, allowAdHoc bool) (*models.Package, error) {
	if pkg, ok := b.FindPackage(name); ok {
		return pkg, nil
	}
	if allowAdHoc && b.IsPackageInstalled(name) {
		logger.Debug("Package %s not found in config; treating as ad-hoc", name)
		return &models.Package{
			Command: name,
		}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrPkgNotFound, name)
}

// guardUninstall aborts if we try to uninstall something not present.
func (b *Base) guardUninstall(isInstalled bool, name string, verb string) error {
	if verb == "uninstall" && !isInstalled {
		logger.Info("Skipping %s: package not installed", name)
		return nil
	}
	return nil
}

// guardUpgrade decides whether an upgrade should run.
func (b *Base) guardUpgrade(isInstalled bool, displayName, execName string) (bool, error) {
	if !isInstalled {
		logger.Info("Skipping %s: package not installed", displayName)
		return false, nil
	}

	state, err := brew.FetchState(b.Runner)
	if err != nil {
		return false, fmt.Errorf("failed to check update status: %w", err)
	}
	if _, out := state.Outdated[execName]; !out {
		logger.Success("%s is already up to date", displayName)
		return false, nil
	}
	return true, nil
}

// handleSelectedPackage executes the given action on a single package.
//
// Parameters:
//   - action: the PackageAction to perform
//   - name: the name of the package to process
//   - validateFunc: function to determine whether the package is valid
//
// Returns:
//   - error: if the operation fails at any step
//
// Behavior:
//   - Validates presence and installation state
//   - Skips upgrade if not outdated
//   - Skips package if already installed and SkipMessage is provided
//   - Runs brew command and updates cache if applicable
func (b *Base) handleSelectedPackage(
	action PackageAction,
	humanName string,
	isValid func(string) bool,
	allowAdHoc bool,
) error {
	// 1. Resolve & canonicalise
	pkg, err := b.resolvePackageScoped(humanName, allowAdHoc)
	if err != nil {
		return err
	}

	execName := b.GetPackageName(pkg)
	installed := b.IsPackageInstalled(execName)

	// 2. Pre-flight guards
	if err := b.guardUninstall(installed, humanName, action.ActionVerb); err != nil {
		return err
	}

	if action.ActionVerb == "upgrade" {
		cont, err := b.guardUpgrade(installed, humanName, execName)
		if err != nil || !cont {
			return err
		}
	}

	if !isValid(execName) {
		logger.Info("Skipping %s: validation failed", humanName)
		return nil
	}

	if installed && action.SkipMessage != "" {
		logger.Success(action.SkipMessage, execName)
		return nil
	}

	// 3. Actual command
	if err := utils.RunBrewCommand(
		b.Runner,
		action.ActionVerb,
		execName,
		[]string{"Warning: The post-install step did not complete successfully"},
	); err != nil {
		return fmt.Errorf("error during %s of %s: %w",
			action.ActionVerb, humanName, err)
	}

	// 4. Post-run housekeeping
	if action.ActionVerb == "upgrade" {
		if _, err := brew.FetchOutdatedPackages(b.Runner); err != nil {
			logger.LogError("Failed to update outdated cache: %v", err)
		}
	}

	b.touchVersionCache(execName)

	logger.Success("%s has been %s successfully!",
		humanName, pastTense[action.ActionVerb])
	return nil
}

func (b *Base) touchVersionCache(execName string) {
	st, err := brew.FetchState(b.Runner)
	if err != nil {
		logger.Debug("versions.Touch skipped: fetch state failed: %v", err)
		return
	}
	v, ok := st.Installed[execName]
	if !ok || v == "" {
		return
	}

	res := versions.NewResolver(b.Runner)
	if err := res.Touch(execName, v); err != nil {
		logger.Debug("versions.Touch failed for %s: %v", execName, err)
	}
}
