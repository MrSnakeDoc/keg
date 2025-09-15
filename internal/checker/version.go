package checker

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = "unknown"
)

func PrintVersion() {
	fmt.Println("Keg - Package installer for development environment")
	fmt.Printf("  %-10s %s\n", "Version:", Version)
	fmt.Printf("  %-10s %s\n", "Go Version:", GoVersion)
	fmt.Printf("  %-10s %s\n", "Git Commit:", Commit)
	fmt.Printf("  %-10s %s\n", "Built:", Date)
	fmt.Printf("  %-10s %s/%s\n", "OS/Arch:", runtime.GOOS, runtime.GOARCH)
}
