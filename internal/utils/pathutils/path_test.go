package pathutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToHomePathFormatRespectsPathBoundaries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	inside := filepath.Join(home, "projects", "keg")
	got, err := ToHomePathFormat(inside)
	if err != nil {
		t.Fatalf("ToHomePathFormat failed: %v", err)
	}
	if want := filepath.Join("~", "projects", "keg"); got != want {
		t.Fatalf("ToHomePathFormat returned %q, want %q", got, want)
	}

	if got, err := ToHomePathFormat(home); err != nil || got != "~" {
		t.Fatalf("home path returned %q, err=%v; want ~", got, err)
	}

	sibling := home + "-backup"
	if got, err := ToHomePathFormat(sibling); err != nil || got != sibling {
		t.Fatalf("sibling path returned %q, err=%v; want unchanged path", got, err)
	}
}

func TestToAbsolutePathOnlyExpandsHomeNotation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got, err := ToAbsolutePath("~"); err != nil || got != home {
		t.Fatalf("home notation returned %q, err=%v; want %q", got, err, home)
	}

	relative := filepath.Join("~", "projects", "keg")
	if got, err := ToAbsolutePath(relative); err != nil || got != filepath.Join(home, "projects", "keg") {
		t.Fatalf("relative home notation returned %q, err=%v", got, err)
	}

	if got, err := ToAbsolutePath("~backup"); err != nil || got != "~backup" {
		t.Fatalf("user-like notation returned %q, err=%v; want unchanged path", got, err)
	}

	if _, err := os.Stat(home); err != nil {
		t.Fatalf("test home disappeared: %v", err)
	}
}
