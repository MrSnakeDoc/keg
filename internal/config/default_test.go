package config

import "testing"

func TestDefaultCheckerConfig(t *testing.T) {
	c := DefaultCheckerConfig()
	if c.CheckFrequency == 0 {
		t.Fatal("want non-zero CheckFrequency")
	}
	if c.ForceBypassSave {
		t.Fatal("want ForceBypassSave=false")
	}
	if c.VersionURL == "" {
		t.Fatal("want VersionURL")
	}
}

func TestDefaultUpdateConfig(t *testing.T) {
	c := DefaultUpdateConfig()
	if c.CheckFrequency != 0 {
		t.Fatal("want CheckFrequency=0")
	}
	if !c.ForceBypassSave {
		t.Fatal("want ForceBypassSave=true")
	}
	if c.VersionURL == "" {
		t.Fatal("want VersionURL")
	}
}
