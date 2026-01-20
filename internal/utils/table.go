package utils

import "github.com/MrSnakeDoc/keg/internal/logger"

type PackageStatus struct {
	Name      string
	Installed string
	Status    string
}

func CreateStatusTable(title string, packages []PackageStatus) {
	if title != "" {
		logger.Info("%s", title)
	}

	table := logger.CreateTable([]string{"Package", "Installed", "Status"})

	for _, pkg := range packages {
		err := table.Append([]string{pkg.Name, pkg.Installed, pkg.Status})
		if err != nil {
			logger.LogError("Error appending to table: %v", err)
			return
		}
	}

	err := table.Render()
	if err != nil {
		logger.LogError("Error rendering table: %v", err)
		return
	}
}
