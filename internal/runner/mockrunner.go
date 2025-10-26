package runner

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type MockRunner struct {
	Commands     []MockCommand
	Responses    map[string]MockResponse
	ResponseFunc func(name string, args ...string) ([]byte, error)
}

type MockCommand struct {
	Name    string
	Args    []string
	Dir     string
	Env     []string
	Timeout time.Duration
	Mode    Mode
}

type MockResponse struct {
	Output []byte
	Error  error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		Commands:  []MockCommand{},
		Responses: make(map[string]MockResponse),
	}
}

func (m *MockRunner) Run(
	ctx context.Context,
	timeout time.Duration,
	mode Mode,
	name string,
	args ...string,
) ([]byte, error) {
	m.Commands = append(m.Commands, MockCommand{
		Name:    name,
		Args:    args,
		Timeout: timeout,
		Mode:    mode,
	})

	key := cmdKey(name, args...)
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Error
	}
	if m.ResponseFunc != nil {
		return m.ResponseFunc(name, args...)
	}
	if mode == Stream {
		return nil, nil
	}
	return []byte{}, nil
}

func (m *MockRunner) AddResponse(key string, output []byte, err error) {
	m.Responses[key] = MockResponse{
		Output: output,
		Error:  err,
	}
}

func (m *MockRunner) Command(name string, args ...string) *exec.Cmd {
	m.Commands = append(m.Commands, MockCommand{
		Name: name,
		Args: args,
	})

	return exec.Command("echo", "MockCommand called but shouldn't be used directly")
}

func (m *MockRunner) CommandWithEnv(name string, env []string, args ...string) *exec.Cmd {
	m.Commands = append(m.Commands, MockCommand{
		Name: name,
		Args: args,
		Env:  env,
	})

	return exec.Command("echo", "MockCommandWithEnv called but shouldn't be used directly")
}

func (m *MockRunner) CommandDir(name string, dir string, args ...string) *exec.Cmd {
	m.Commands = append(m.Commands, MockCommand{
		Name: name,
		Args: args,
		Dir:  dir,
	})

	return exec.Command("echo", "MockCommandDir called but shouldn't be used directly")
}

func (m *MockRunner) Output(name string, args ...string) ([]byte, error) {
	m.Commands = append(m.Commands, MockCommand{
		Name: name,
		Args: args,
	})

	key := cmdKey(name, args...)
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Error
	}

	if m.ResponseFunc != nil {
		return m.ResponseFunc(name, args...)
	}

	return []byte{}, nil
}

func (m *MockRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return m.Output(name, args...)
}

func cmdKey(name string, args ...string) string {
	key := name
	for _, arg := range args {
		key += "|" + arg
	}
	return key
}

func (m *MockRunner) VerifyCommand(name string, args ...string) bool {
	for _, cmd := range m.Commands {
		if cmd.Name == name && argsEqual(cmd.Args, args) {
			return true
		}
	}
	return false
}

func (m *MockRunner) VerifyRunCount(name string, count int) bool {
	runCount := 0
	for _, cmd := range m.Commands {
		if cmd.Name == name {
			runCount++
		}
	}
	return runCount == count
}

func argsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *MockRunner) GetBrewList(packages ...string) {
	var output string
	for _, pkg := range packages {
		output += pkg + "\n"
	}
	m.AddResponse("brew|list", []byte(output), nil)
}

func (m *MockRunner) GetBrewOutdated(packages ...string) {
	var output string
	for _, pkg := range packages {
		output += fmt.Sprintf("%s 1.0.0 < 2.0.0\n", pkg)
	}
	m.AddResponse("brew|outdated", []byte(output), nil)
}

func (m *MockRunner) MockBrewInfoV2Formula(name, installed, stable string) {
	payload := []byte(fmt.Sprintf(`{
		"formulae": [{
			"name": "%s",
			"versions": {"stable": "%s"},
			"installed": [{"version": "%s"}]
		}]
	}`, name, stable, installed))
	m.AddResponse("brew|info|--json=v2|"+name, payload, nil)
}
