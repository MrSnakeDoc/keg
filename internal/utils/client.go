package utils

import (
	"io"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

func Try(f func() error) {
	if err := f(); err != nil {
		logger.LogError("deferred cleanup failed: %v", err)
	}
}

func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		logger.LogError("close failed: %v", err)
	}
}

func DefaultIfNil(value, defaultValue interface{}) interface{} {
	if value == nil {
		return defaultValue
	}
	return value
}
