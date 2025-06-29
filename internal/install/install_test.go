package install

import (
	"fmt"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/prompter"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

type mockPrompter struct {
	confirms []bool
	prompts  []string
	err      error

	ci, pi int
}

func (m *mockPrompter) Confirm(string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	if m.ci >= len(m.confirms) {
		return false, fmt.Errorf("unexpected Confirm call")
	}
	res := m.confirms[m.ci]
	m.ci++
	return res, nil
}

func (m *mockPrompter) Prompt(string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.pi >= len(m.prompts) {
		return "", fmt.Errorf("unexpected Prompt call")
	}
	res := m.prompts[m.pi]
	m.pi++
	return res, nil
}

var executeTestCases = []struct {
	name          string
	args          []string
	all           bool
	mockError     error
	interactive   bool
	prompter      *mockPrompter
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
	{
		name:        "Interactive add missing package",
		args:        []string{"newpkg"},
		all:         false,
		interactive: true,
		prompter: &mockPrompter{
			confirms: []bool{true, false}, // add? yes, optional? no
			prompts:  []string{""},        // binary name (Empty => same as command)
		},
	},
	{
		name:        "Interactive skip missing package",
		args:        []string{"skipme"},
		all:         false,
		interactive: true,
		prompter: &mockPrompter{
			confirms: []bool{false}, // add? no
		},
	},
}

func TestInstaller_Execute(t *testing.T) {
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

			p := prompter.Prompter(tt.prompter)

			installer := New(config, mockRunner, p)

			err := installer.Execute(tt.args, tt.all, tt.interactive)

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

			if tt.interactive && len(tt.args) == 1 && tt.args[0] == "newpkg" {
				found := false
				for _, p := range config.Packages {
					if p.Command == "newpkg" {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("interactive add failed: package not appended")
				}
			}
		})
	}
}
