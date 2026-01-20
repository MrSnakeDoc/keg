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

// UseMiddlewareChain wraps a CommandFactory with a chain of middlewares.
// Optimized: Pre-stores the middleware slice to avoid repeated varargs expansion.
func UseMiddlewareChain(middlewares ...MiddlewareFunc) MiddlewareChain {
	// Pre-allocate: copy middlewares slice once at construction time
	mwCopy := make([]MiddlewareFunc, len(middlewares))
	copy(mwCopy, middlewares)
	mwLen := len(mwCopy)

	return func(factory CommandFactory) CommandFactory {
		return func() *cobra.Command {
			cmd := factory()
			orig := cmd.PreRunE

			// Inject a PreRunE that executes the middleware chain
			cmd.PreRunE = func(c *cobra.Command, a []string) error {
				// Fast path: no middlewares
				if mwLen == 0 {
					if orig != nil {
						return orig(c, a)
					}
					return nil
				}

				// Execute middleware chain
				// We still need closures here (it's the nature of middleware pattern)
				// but we minimize allocations by reusing the pre-stored slice
				var chain func(int) error
				chain = func(i int) error {
					if i >= mwLen {
						if orig != nil {
							return orig(c, a)
						}
						return nil
					}
					return mwCopy[i](c, a, func(nc *cobra.Command, na []string) error {
						return chain(i + 1)
					})
				}
				return chain(0)
			}
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
