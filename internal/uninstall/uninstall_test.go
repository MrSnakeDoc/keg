package uninstall

import (
	"testing"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

// Table-driven test cases for Uninstaller.Execute
var executeTestCases = []struct {
	name          string
	args          []string
	all           bool
	remove        bool
	force         bool
	expectedError string
}{
	{
		name: "Uninstall specific installed package",
		args: []string{"pkg1"},
	},
	{
		name: "Uninstall all installed packages",
		all:  true,
	},
	{
		name:   "Uninstall with remove flag (specific)",
		args:   []string{"pkg2"},
		remove: true,
	},
	{
		name:   "Uninstall with remove and all",
		all:    true,
		remove: true,
		force:  true,
	},
}

func TestUninstaller_Execute(t *testing.T) {
	oldSave := saveConfig
	saveConfig = func(_ *models.Config) error { return nil }
	defer func() { saveConfig = oldSave }()

	for _, tt := range executeTestCases {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := runner.NewMockRunner()
			mockRunner.GetBrewList("pkg1")

			config := &models.Config{
				Packages: []models.Package{
					{Command: "pkg1"},
					{Command: "pkg2"},
					{Command: "pkg3", Optional: true},
				},
			}

			uninstaller := New(config, mockRunner)

			err := uninstaller.Execute(tt.args, tt.all, tt.remove, tt.force)

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
		})
	}
}
