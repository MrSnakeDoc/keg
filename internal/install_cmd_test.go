package internal

import (
	"strings"
	"testing"
)

func TestInstallCmd_FlagValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "No args without --all",
			args:          []string{"install"},
			expectedError: "Missing targets: provide package names or use --all",
		},
		{
			name:          "--all with specific packages",
			args:          []string{"install", "foo", "--all"},
			expectedError: "Invalid flag combination: cannot use --all with named packages",
		},
		{
			name:          "--all with --add",
			args:          []string{"install", "--all", "--add"},
			expectedError: "Invalid flag combination: cannot combine --all with --add",
		},
		{
			name:          "--optional without --add",
			args:          []string{"install", "foo", "--optional"},
			expectedError: "Invalid flag combination: --optional and --binary require --add",
		},
		{
			name:          "--binary without --add",
			args:          []string{"install", "foo", "--binary", "bar"},
			expectedError: "Invalid flag combination: --optional and --binary require --add",
		},
		{
			name:          "--binary with multiple packages",
			args:          []string{"install", "foo", "bar", "--add", "--binary", "baz"},
			expectedError: "Invalid usage: --binary can only be used with a single package",
		},
	}

	root := NewRootCmd()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root.SetArgs(tt.args)
			_, err := root.ExecuteC()

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !strings.Contains(err.Error(), "already logged") {
				t.Errorf("expected sentinel error, got: %v", err)
			}
		})
	}
}
