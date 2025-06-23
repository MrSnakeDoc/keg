package install

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

var executeTestCases = []struct {
	name          string
	args          []string
	all           bool
	mockError     error
	expectedError string
}{
	{
		name: "No args standard execution",
		args: []string{},
		all:  false,
	},
	{
		name: "With args",
		args: []string{"pkg1", "pkg2"},
		all:  false,
	},
	{
		name: "With all flag",
		args: []string{},
		all:  true,
	},
	{
		name:          "Error when using --all with specific packages",
		args:          []string{"pkg1"},
		all:           true,
		expectedError: "you cannot use --all with specific packages",
	},
}

func TestInstaller_Execute(t *testing.T) {
	for _, tt := range executeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := runner.NewMockRunner()

			mockRunner.GetBrewList("pkg1")

			config := &models.Config{
				Packages: []models.Package{
					{Command: "pkg1", Optional: false},
					{Command: "pkg2", Optional: false},
					{Command: "pkg3", Optional: true},
				},
			}

			installer := New(config, mockRunner)

			err := installer.Execute(tt.args, tt.all)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error: %s, got: nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError {
					t.Errorf("Expected error: %s, got: %s", tt.expectedError, err.Error())
				}
				return
			} else if err != nil {
				t.Errorf("Unexpected error: %s", err)
				return
			}

			// Display all executed commands for debugging
			t.Logf("Commands executed: %v", formatCommands(mockRunner.Commands))

			// Verify executed commands
			switch {
			case len(tt.args) > 0:
				verifySpecificPackagesInstalled(t, mockRunner.Commands, tt.args)
			case tt.all:
				verifyAllPackagesInstalled(t, mockRunner.Commands)
			default:
				verifyRequiredPackagesInstalled(t, mockRunner.Commands)
			}
		})
	}
}

func TestNew(t *testing.T) {
	config := &models.Config{}
	mockRunner := runner.NewMockRunner()

	installer := New(config, mockRunner)

	if installer == nil {
		t.Fatal("New returned nil installer")
	}
}

func TestErrorHandling(t *testing.T) {
	mockRunner := runner.NewMockRunner()

	mockRunner.AddResponse("brew|install|failing-pkg", []byte{}, fmt.Errorf("mock installation error"))

	config := &models.Config{
		Packages: []models.Package{
			{Command: "failing-pkg", Optional: false},
		},
	}

	installer := New(config, mockRunner)

	err := installer.Execute([]string{}, false)

	if err == nil {
		t.Fatal("Expected error when brew install fails, got nil")
	}

	if !strings.Contains(err.Error(), "failing-pkg") || !strings.Contains(err.Error(), "mock installation error") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func formatCommands(commands []runner.MockCommand) string {
	result := make([]string, len(commands))
	for i, cmd := range commands {
		args := strings.Join(cmd.Args, " ")
		result[i] = fmt.Sprintf("%s %s (timeout: %v)", cmd.Name, args, cmd.Timeout)
	}
	return strings.Join(result, "\n")
}

func verifySpecificPackagesInstalled(t *testing.T, commands []runner.MockCommand, packages []string) {
	for _, pkg := range packages {
		if pkg != "pkg1" {
			found := false
			for _, cmd := range commands {
				if cmd.Name == "brew" && len(cmd.Args) >= 2 && cmd.Args[0] == "install" && contains(cmd.Args, pkg) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected 'brew install' command with '%s' to be called", pkg)
			}
		}
	}
}

func verifyAllPackagesInstalled(t *testing.T, commands []runner.MockCommand) {
	for _, pkg := range []string{"pkg2", "pkg3"} {
		found := false
		for _, cmd := range commands {
			if cmd.Name == "brew" && len(cmd.Args) >= 2 && cmd.Args[0] == "install" && contains(cmd.Args, pkg) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'brew install' command with '%s' to be called", pkg)
		}
	}
}

func verifyRequiredPackagesInstalled(t *testing.T, commands []runner.MockCommand) {
	found := false
	for _, cmd := range commands {
		if cmd.Name == "brew" && len(cmd.Args) >= 2 && cmd.Args[0] == "install" && contains(cmd.Args, "pkg2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'brew install pkg2' to be called")
	}

	for _, cmd := range commands {
		if cmd.Name == "brew" && len(cmd.Args) >= 2 && cmd.Args[0] == "install" && contains(cmd.Args, "pkg3") {
			t.Errorf("Unexpected 'brew install pkg3' call - pkg3 is optional")
			break
		}
	}
}
