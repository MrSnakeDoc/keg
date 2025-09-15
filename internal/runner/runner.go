package runner

import (
	"context"
	"os"
	"os/exec"
	"time"
)

type Mode int

const (
	Capture Mode = iota
	Stream
)

type CommandRunner interface {
	Run(ctx context.Context, timeout time.Duration, mode Mode,
		name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(
	parent context.Context,
	timeout time.Duration,
	mode Mode,
	name string,
	args ...string,
) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	switch mode {
	case Stream:
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		return nil, cmd.Run()
	default:
		out, err := cmd.CombinedOutput()
		return out, err
	}
}
