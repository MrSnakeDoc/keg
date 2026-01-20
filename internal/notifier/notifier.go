package notifier

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/config"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

const (
	borderColor = "\033[38;5;39m"
	resetColor  = "\033[0m"
	padding     = 2
)

// DisplayUpdateNotification checks for update information and displays a notification if an update is available
func DisplayUpdateNotification() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Error getting user home directory: %v\n", err)
		return
	}

	stateFile := filepath.Join(home, ".local", "state", "keg", "update-check.json")

	if ok, _ := utils.FileExists(stateFile); !ok {
		fmt.Printf("❌ Update state file does not exist: %s\n", stateFile)
		return
	}

	var state config.UpdateState
	if err := utils.FileReader(stateFile, "json", &state); err != nil {
		logger.Debug("failed to load update state: %v", err)
		return
	}

	if !state.UpdateAvailable {
		return
	}

	DisplayVersionUpdate(state.LatestVersion)
}

// DisplayVersionUpdate shows a formatted notification for a new version
func DisplayVersionUpdate(version string) {
	p := printer.NewColorPrinter()

	title := p.Success("New Version Available!")
	detected := p.Info("New version detected:")
	command := p.Warning("Run ")
	updateCmd := p.Success("keg update")
	instruction := p.Warning(" to update.")
	actualVersion := p.Error(checker.Version)
	versionInfo := p.Success(version)

	lines := []string{
		title,
		fmt.Sprintf("%s %s -> %s", detected, actualVersion, versionInfo),
		fmt.Sprintf("%s%s%s", command, updateCmd, instruction),
	}

	maxWidth := utils.GetMaxWidth(lines) + padding*2
	topBottomBorder := borderColor + "╭" + strings.Repeat("─", maxWidth) + "╮" + resetColor
	sideBorder := borderColor + "│" + resetColor

	fmt.Println(topBottomBorder)
	for _, line := range lines {
		paddingLeft := (maxWidth - len(utils.StripANSI(line))) / 2
		paddingRight := maxWidth - len(utils.StripANSI(line)) - paddingLeft
		fmt.Printf("%s%s%s%s%s\n", sideBorder, strings.Repeat(" ", paddingLeft), line, strings.Repeat(" ", paddingRight), sideBorder)
	}
	fmt.Println(borderColor + "╰" + strings.Repeat("─", maxWidth) + "╯" + resetColor)
}
