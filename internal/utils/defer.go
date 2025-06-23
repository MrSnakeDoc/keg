package utils

import (
	"io"
	"log"
	"os"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

func MustSet(key, val string) (old string) {
	old, _ = os.LookupEnv(key)
	if err := os.Setenv(key, val); err != nil {
		logger.LogError("envutil: " + err.Error())
	}
	return
}

func DeferRestore(key, val string) {
	if err := os.Setenv(key, val); err != nil {
		logger.LogError("envutil: impossible de restaurer %s: %v", key, err)
	}
}

func MustClose(c io.Closer) {
	if err := c.Close(); err != nil {
		logger.LogError("closeutil: " + err.Error())
	}
}

func DeferClose(c io.Closer) func() {
	return func() {
		if err := c.Close(); err != nil {
			log.Printf("closeutil: impossible de fermer : %v", err)
		}
	}
}
