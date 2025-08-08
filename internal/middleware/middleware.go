package middleware

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	CtxKeyConfig  contextKey = "configpkg"
	CtxKeyPConfig contextKey = "persistent_config"
)

type CommandFactory func() *cobra.Command

type MiddlewareFunc func(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error

type MiddlewareChain func(factory CommandFactory) CommandFactory

type contextKey string

// func UseMiddlewareChain(middlewares ...MiddlewareFunc) MiddlewareChain {
// 	return func(factory CommandFactory) CommandFactory {
// 		return func() *cobra.Command {
// 			cmd := factory()

// 			original := cmd.PreRunE

// 			cmd.PreRunE = func(c *cobra.Command, args []string) error {
// 				var chain func(cmd *cobra.Command, args []string) error
// 				chain = func(cmd *cobra.Command, args []string) error {
// 					if original != nil {
// 						return original(cmd, args)
// 					}
// 					return nil
// 				}

// 				for i := len(middlewares) - 1; i >= 0; i-- {
// 					mw := middlewares[i]
// 					next := chain
// 					chain = func(cmd *cobra.Command, args []string) error {
// 						return mw(cmd, args, next)
// 					}
// 				}

// 				return chain(c, args)
// 			}

// 			return cmd
// 		}
// 	}
// }

func UseMiddlewareChain(middlewares ...MiddlewareFunc) MiddlewareChain {
	return func(factory CommandFactory) CommandFactory {
		return func() *cobra.Command {
			cmd := factory()
			orig := cmd.PreRunE

			var chain func(i int, c *cobra.Command, a []string) error
			chain = func(i int, c *cobra.Command, a []string) error {
				if i >= len(middlewares) {
					if orig != nil {
						return orig(c, a)
					}
					return nil
				}
				return middlewares[i](c, a, func(nc *cobra.Command, na []string) error {
					return chain(i+1, nc, na)
				})
			}

			cmd.PreRunE = func(c *cobra.Command, a []string) error { return chain(0, c, a) }
			return cmd
		}
	}
}

func Get[T any](cmd *cobra.Command, key contextKey) (T, error) {
	var zero T

	ctx := cmd.Context()
	if ctx == nil {
		return zero, fmt.Errorf("command context is nil")
	}

	val := ctx.Value(key)
	if val == nil {
		return zero, fmt.Errorf("context value %q is nil", key)
	}

	casted, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("context value %q has wrong type: %T", key, val)
	}

	return casted, nil
}
