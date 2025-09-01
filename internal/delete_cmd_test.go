package internal

import (
	"strings"
	"testing"
)

func TestDeleteCmd_FlagValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "No args without --all",
			args:          []string{"delete"},
			expectedError: "Missing targets: provide package names or use --all",
		},
		{
			name:          "--all with specific packages",
			args:          []string{"delete", "foo", "--all"},
			expectedError: "Invalid flag combination: cannot use --all with named packages",
		},
		{
			name:          "--all with --remove without --force",
			args:          []string{"delete", "--all", "--remove"},
			expectedError: "--all with --remove requires --force",
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
