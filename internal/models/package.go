package models

type Package struct {
	Command  string `yaml:"command"`
	Binary   string `yaml:"binary,omitempty"`
	Optional bool   `yaml:"optional,omitempty"`
}

type Config struct {
	Packages []Package `yaml:"packages"`
}
