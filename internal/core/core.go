package core

import (
	"context"
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
//   - cache: Unified cache for all brew operations
//   - Runner: A CommandRunner instance to execute system commands
//
// It stores the user configuration, uses the unified cache,
// and interacts with the underlying system via CommandRunner.
type Base struct {
	Config       *models.Config
	cache        *brew.UnifiedCache
	Runner       runner.CommandRunner
	upgradedPkgs []string
}

// BrewSessionState holds a snapshot of brew's view of the world for a
// single HandlePackages run. It is intentionally small and read-only.
type BrewSessionState struct {
	State *brew.BrewState
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
	if r == nil {
		r = &runner.ExecRunner{}
	}

	cache, err := brew.GetCache(r)
	if err != nil {
		logger.Debug("failed to initialize unified cache: %v", err)
		// Fallback to empty cache
		cache = &brew.UnifiedCache{}
	}

	return &Base{
		Config: config,
		cache:  cache,
		Runner: r,
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
//   - This function uses the unified cache which auto-refreshes when stale.
func (b *Base) IsPackageInstalled(name string) bool {
	if b.cache == nil {
		return false
	}
	return b.cache.IsInstalled(name)
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

// loadSessionState initializes a BrewSessionState for operations that need a
// global view of installed/outdated packages (typically upgrades).
func (b *Base) loadSessionState() (*BrewSessionState, error) {
	st, err := brew.FetchState(b.Runner)
	if err != nil {
		return nil, err
	}
	return &BrewSessionState{State: st}, nil
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
	// Ensure finalizeUpgrades always runs for upgrades, even on early returns
	if opts.Action.ActionVerb == "upgrade" {
		defer b.finalizeUpgrades()
	}

	// Preload a shared brew session for actions that care about global state.
	var session *BrewSessionState
	if opts.Action.ActionVerb == "upgrade" {
		var err error
		session, err = b.loadSessionState()
		if err != nil {
			return fmt.Errorf("failed to load brew state: %w", err)
		}
	}

	if opts.FilterFunc == nil {
		opts.FilterFunc = func(*models.Package) bool { return true }
	}
	if opts.ValidateFunc == nil {
		opts.ValidateFunc = func(string) bool { return true }
	}

	if len(opts.Packages) > 0 {
		for _, pkgName := range opts.Packages {
			if err := b.handleSelectedPackageWithSession(opts.Action, pkgName, opts.ValidateFunc, opts.AllowAdHoc, session); err != nil {
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
		if err := b.handleSelectedPackageWithSession(opts.Action, name, opts.ValidateFunc, opts.AllowAdHoc, session); err != nil {
			return fmt.Errorf("failed to %s package %s: %w", opts.Action.ActionVerb, name, err)
		}
	}
	return nil
}

func (b *Base) finalizeUpgrades() {
	if len(b.upgradedPkgs) == 0 {
		return
	}

	logger.Info("Updating version cache for upgraded packages...")

	// dedupe to avoid double work
	seen := make(map[string]struct{}, len(b.upgradedPkgs))
	uniq := make([]string, 0, len(b.upgradedPkgs))
	for _, n := range b.upgradedPkgs {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		uniq = append(uniq, n)
	}

	// 3) bulk refresh version cache per upgraded package (after cleanup)
	b.bulkTouchVersionCache(uniq)

	// reset for next HandlePackages run
	b.upgradedPkgs = b.upgradedPkgs[:0]
}

// bulkTouchVersionCache updates the versions cache for a list of upgraded
// packages in a single pass, avoiding redundant brew and resolver calls.
func (b *Base) bulkTouchVersionCache(names []string) {
	if len(names) == 0 {
		return
	}

	res := versions.NewResolver(b.Runner)

	// 1) Snapshot before (likely stale) for all names
	before, err := res.ResolveBulk(context.Background(), names)
	if err != nil {
		logger.Debug("bulk versions.ResolveBulk (before) failed: %v", err)
		return
	}

	// 2) Evict and re-resolve to force fresh reads from brew
	for _, n := range names {
		_ = res.Remove(n)
	}
	after, err := res.ResolveBulk(context.Background(), names)
	if err != nil {
		logger.Debug("bulk versions.ResolveBulk (fresh) failed: %v", err)
		return
	}

	// 3) Only write if value actually changed (avoid useless writes)
	for _, n := range names {
		fresh, ok := after[n]
		if !ok || fresh.Installed == "" {
			continue
		}
		prev := before[n]
		if fresh.Installed == prev.Installed {
			continue
		}
		if err := res.Touch(n, fresh.Installed); err != nil {
			logger.Debug("versions.Touch failed for %s: %v", n, err)
		}
	}
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

func (b *Base) guardUpgrade(session *BrewSessionState, isInstalled bool, displayName, execName string) bool {
	if !isInstalled {
		logger.Info("Skipping %s: package not installed", displayName)
		return false
	}

	if session == nil || session.State == nil {
		// Without a session, we conservatively attempt the upgrade.
		return true
	}
	if _, out := session.State.Outdated[execName]; !out {
		logger.Success("%s is already up to date", displayName)
		return false
	}
	return true
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
	return b.handleSelectedPackageWithSession(action, humanName, isValid, allowAdHoc, nil)
}

// handleSelectedPackageWithSession is the internal implementation that optionally
// receives a BrewSessionState to avoid repeated global brew calls.
func (b *Base) handleSelectedPackageWithSession(
	action PackageAction,
	humanName string,
	isValid func(string) bool,
	allowAdHoc bool,
	session *BrewSessionState,
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
		if !b.guardUpgrade(session, installed, humanName, execName) {
			return nil
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

	// Post-run bookkeeping per action
	switch action.ActionVerb {
	case "upgrade":
		// Mark as upgraded for bulk finalization
		b.upgradedPkgs = append(b.upgradedPkgs, execName)
		// Update cache immediately
		if b.cache != nil {
			_ = b.cache.MarkUpgraded(execName, "") // version will be fetched in finalize
		}

	case "install":
		// Update unified cache to reflect installation
		if b.cache != nil {
			_ = b.cache.MarkInstalled(execName, "")
		}
		b.touchVersionCache(execName) // force resolver to record the installed version

	case "uninstall":
		// Update unified cache to reflect removal
		if b.cache != nil {
			_ = b.cache.MarkUninstalled(execName)
		}
		err = versions.NewResolver(b.Runner).Remove(execName)
		if err != nil {
			logger.Debug("versions.Remove failed for %s: %v", execName, err)
		}
	}

	logger.Success("%s has been %s successfully!",
		humanName, pastTense[action.ActionVerb])
	return nil
}

// touchVersionCache updates the versions cache for the given executable.
//
// Parameters:
//   - execName: the name of the executable/package to update in the cache
//
// Behavior:
//   - If the package is no longer installed, it removes it from the cache
//   - Otherwise, it refreshes the cached version if it has changed
func (b *Base) touchVersionCache(execName string) {
	res := versions.NewResolver(b.Runner)

	before, err := res.ResolveBulk(context.Background(), []string{execName})
	if err != nil {
		logger.Debug("versions.ResolveBulk (before) failed for %s: %v", execName, err)
		return
	}
	prev := before[execName]

	_ = res.Remove(execName)
	after, err := res.ResolveBulk(context.Background(), []string{execName})
	if err != nil {
		logger.Debug("versions.ResolveBulk (fresh) failed for %s: %v", execName, err)
		return
	}
	fresh, ok := after[execName]
	if !ok || fresh.Installed == "" {
		logger.Debug("versions.ResolveBulk (fresh) empty for %s", execName)
		return
	}

	if fresh.Installed != prev.Installed {
		if err := res.Touch(execName, fresh.Installed); err != nil {
			logger.Debug("versions.Touch failed for %s: %v", execName, err)
		}
	}
}
