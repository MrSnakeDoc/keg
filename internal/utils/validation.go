package utils

import "fmt"

func ValidateBinaryArgs(args []string, binary string) error {
	if binary != "" && len(args) > 1 {
		return fmt.Errorf("--binary flag can only be used when adding a single package")
	}
	return nil
}
