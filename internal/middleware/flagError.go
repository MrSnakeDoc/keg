package middleware

import (
	"errors"

	"github.com/MrSnakeDoc/keg/internal/errs"
	"github.com/MrSnakeDoc/keg/internal/logger"
)

var ErrLogged = errors.New("already logged")

func FlagComboError(code errs.Code, a ...any) error {
	msg := errs.Msg(code, a...)
	logger.LogError("%s", msg)
	return ErrLogged
}
