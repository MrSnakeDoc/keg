package utils

import (
	"os"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

func MustSet(key, val string) (old string) {
	old, _ = os.LookupEnv(key)
	if err := os.Setenv(key, val); err != nil {
		logger.LogError("envutil: " + err.Error())
	}
	return old
}

func DeferRestore(key, val string) {
	if err := os.Setenv(key, val); err != nil {
		logger.LogError("envutil: impossible de restaurer %s: %v", key, err)
	}
}
