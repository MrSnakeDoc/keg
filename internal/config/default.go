package config

import "time"

type Config struct {
	VersionURL      string
	ChecksumBaseURL string
	CheckFrequency  time.Duration
	ForceBypassSave bool
}

type UpdateState struct {
	LastChecked     time.Time `json:"last_checked"`
	UpdateAvailable bool      `json:"update_available,omitempty"`
	LatestVersion   string    `json:"latest_version,omitempty"`
}

func baseConfig() Config {
	return Config{
		VersionURL:      "https://api.github.com/repos/MrSnakeDoc/keg/releases/latest",
		ChecksumBaseURL: "",
	}
}

func DefaultCheckerConfig() Config {
	config := baseConfig()
	config.CheckFrequency = 24 * time.Hour
	config.ForceBypassSave = false
	return config
}

func DefaultUpdateConfig() Config {
	config := baseConfig()
	config.CheckFrequency = 0
	config.ForceBypassSave = true
	return config
}
