package middleware

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	CtxKeyConfig contextKey = "config"
)

type CommandFactory func() *cobra.Command

type MiddlewareFunc func(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error

type MiddlewareChain func(factory CommandFactory) CommandFactory

type contextKey string

func UseMiddlewareChain(middlewares ...MiddlewareFunc) MiddlewareChain {
	return func(factory CommandFactory) CommandFactory {
		return func() *cobra.Command {
			cmd := factory()

			original := cmd.PreRunE

			cmd.PreRunE = func(c *cobra.Command, args []string) error {
				var chain func(cmd *cobra.Command, args []string) error
				chain = func(cmd *cobra.Command, args []string) error {
					if original != nil {
						return original(cmd, args)
					}
					return nil
				}

				for i := len(middlewares) - 1; i >= 0; i-- {
					mw := middlewares[i]
					next := chain
					chain = func(cmd *cobra.Command, args []string) error {
						return mw(cmd, args, next)
					}
				}

				return chain(c, args)
			}

			return cmd
		}
	}
}

func Get[T any](cmd *cobra.Command, key contextKey) (T, error) {
	val := cmd.Context().Value(key)
	if val == nil {
		var zero T
		return zero, fmt.Errorf("context value %q is nil", key)
	}

	casted, ok := val.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("context value %q has wrong type", key)
	}

	return casted, nil
}
