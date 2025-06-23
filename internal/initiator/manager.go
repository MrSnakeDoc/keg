package initiator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Initiator struct {
	ConfigPath string
}

func New() *Initiator {
	return &Initiator{}
}

func (*Initiator) Execute() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkgFile := filepath.Join(cwd, "keg.yml")
	if ok, _ := utils.FileExists(pkgFile); !ok {
		if err := utils.CreateFile(pkgFile, []byte("packages: []\n"), "yaml", 0o644); err != nil {
			return err
		}
		logger.Success("Created empty %s file", pkgFile)
	}

	_, err = utils.EnsureUpdateStateFileExists()
	if err != nil {
		logger.Debug("Failed to ensure update state file exists: %v", err)
		return fmt.Errorf("failed to ensure update state file exists: %w", err)
	}

	cfg := &globalconfig.PersistentConfig{
		PackagesFile: pkgFile,
	}

	err = cfg.Save()
	if err != nil {
		return err
	}

	return nil
}
