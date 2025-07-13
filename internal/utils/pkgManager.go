package utils

import "fmt"

type pmCmd struct {
	Install []string
	Update  []string
}

var managers = map[string]pmCmd{
	"apt": {
		Install: []string{"apt", "install", "-y"},
		Update:  []string{"bash", "-c", "sudo apt update && sudo apt upgrade -y"},
	},
	"dnf": {
		Install: []string{"dnf", "install", "-y"},
		Update:  []string{"dnf", "upgrade", "--refresh", "-y"},
	},
	"pacman": {
		Install: []string{"pacman", "-S", "--noconfirm"},
		Update:  []string{"pacman", "-Syu", "--noconfirm"},
	},
}

func PackageManager() (pmCmd, error) {
	for _, cmd := range managers {
		if CommandExists(cmd.Install[0]) {
			return cmd, nil
		}
	}
	return pmCmd{}, fmt.Errorf("no supported package manager found")
}
