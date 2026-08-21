package utils

import (
	"fmt"
	"strings"
	"testing"
)

func validVersionInfo() VersionInfo {
	return VersionInfo{
		Version: "1.2.3",
		URL:     fmt.Sprintf("https://github.com/MrSnakeDoc/keg/releases/download/v1.2.3/%s", AssetName("1.2.3")),
		SHA256:  strings.Repeat("a", 64),
	}
}

func TestValidateVersionAcceptsValidMetadata(t *testing.T) {
	info := validVersionInfo()
	if err := ValidateVersion(&info); err != nil {
		t.Fatalf("ValidateVersion rejected valid metadata: %v", err)
	}
}

func TestValidateVersionRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*VersionInfo)
	}{
		{name: "nil info", mutate: nil},
		{name: "invalid version", mutate: func(info *VersionInfo) { info.Version = "1.2" }},
		{name: "invalid checksum length", mutate: func(info *VersionInfo) { info.SHA256 = "deadbeef" }},
		{name: "invalid checksum characters", mutate: func(info *VersionInfo) { info.SHA256 = strings.Repeat("z", 64) }},
		{name: "insecure URL", mutate: func(info *VersionInfo) { info.URL = strings.Replace(info.URL, "https://", "http://", 1) }},
		{name: "wrong host", mutate: func(info *VersionInfo) { info.URL = strings.Replace(info.URL, "github.com", "example.com", 1) }},
		{name: "wrong asset", mutate: func(info *VersionInfo) { info.URL = strings.Replace(info.URL, AssetName(info.Version), "other", 1) }},
		{name: "query string", mutate: func(info *VersionInfo) { info.URL += "?download=1" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mutate == nil {
				if err := ValidateVersion(nil); err == nil {
					t.Fatal("ValidateVersion accepted nil metadata")
				}
				return
			}

			info := validVersionInfo()
			tt.mutate(&info)
			if err := ValidateVersion(&info); err == nil {
				t.Fatal("ValidateVersion accepted invalid metadata")
			}
		})
	}
}
