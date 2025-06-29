package install

import (
	"fmt"
	"os"

	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/prompter"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Installer struct {
	*core.Base
	prompt prompter.Prompter
}

func New(config *models.Config, r runner.CommandRunner, p prompter.Prompter) *Installer {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	if p == nil { // création lazy
		p = prompter.New(os.Stdin, os.Stdout)
	}

	return &Installer{
		Base:   core.NewBase(config, r),
		prompt: p,
	}
}

func (i *Installer) Execute(args []string, all bool, interactive bool) error {
	// Check if we need interactive package addition
	if interactive && len(args) > 0 {
		var err error
		args, err = i.handleMissingPackages(args)
		if err != nil {
			return err
		}
		if len(args) == 0 && !all {
			return nil
		}
	}

	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:        "Installing",
		ActionVerb:  "install",
		SkipMessage: "%s is already installed",
	})

	if len(args) > 0 && all {
		return fmt.Errorf("you cannot use --all with specific packages")
	}

	if len(args) > 0 {
		opts.Packages = args
	}

	if all {
		opts.FilterFunc = func(_ *models.Package) bool { return true }
	}

	return i.HandlePackages(opts)
}

// handleMissingPackages checks if packages exist and interactively adds them if needed
func (i *Installer) handleMissingPackages(pkgs []string) ([]string, error) {
	filtered := pkgs[:0]
	anyAdded := false

	for _, name := range pkgs {
		if _, found := i.FindPackage(name); found {
			filtered = append(filtered, name)
			continue
		}

		if i.promptAndAddPackage(name) {
			filtered = append(filtered, name)
			anyAdded = true
		}
	}

	if anyAdded {
		if err := utils.SaveConfig(i.Config); err != nil {
			return nil, err
		}
	}

	return filtered, nil
}

// promptAndAddPackage asks the user if they want to add a missing package
func (i *Installer) promptAndAddPackage(name string) bool {
	logger.Info("Package '%s' not found in your config.", name)

	ok, err := i.prompt.Confirm("Do you want to add it now?")
	if err != nil {
		logger.LogError("Prompt failed: %v", err)
		return false
	}
	if !ok {
		logger.Info("Skipping addition of '%s'", name)
		return false
	}

	binary, err := i.prompt.Prompt(fmt.Sprintf("Binary name (leave empty if same as '%s'): ", name))
	if err != nil {
		logger.LogError("Prompt failed: %v", err)
		return false
	}
	if binary == "" {
		binary = name
	}

	optional, err := i.prompt.Confirm("Is this an optional package?")
	if err != nil {
		logger.LogError("Prompt failed: %v", err)
		return false
	}

	// Add to config
	i.Config.Packages = append(i.Config.Packages, models.Package{
		Command:  name,
		Binary:   binary,
		Optional: optional,
	})

	logger.Success("Added '%s' to your config", name)
	return true
}
