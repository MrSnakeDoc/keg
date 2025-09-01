package errs

import "fmt"

type Code string

const (
	AllWithNamedPackages    Code = "ALL_WITH_NAMED_PACKAGES"
	ProvidePkgsOrAll        Code = "PROVIDE_PKGS_OR_ALL"
	AllWithRemoveNeedsForce Code = "ALL_WITH_REMOVE_NEEDS_FORCE"
	AllWithAddInvalid       Code = "ALL_WITH_ADD_INVALID"
	OptOrBinRequireAdd      Code = "OPT_OR_BIN_REQUIRE_ADD"
	BinarySinglePackageOnly Code = "BINARY_SINGLE_PACKAGE_ONLY"
)

var messages = map[Code]string{
	AllWithNamedPackages: `Invalid flag combination: cannot use --all with named packages

Usage:
  - %[1]s everything listed in keg.yml:
      keg %[2]s --all
  - %[1]s only specific packages:
      keg %[2]s foo bar%[3]s

Reason:
  --all targets everything, named args target a subset.`,

	ProvidePkgsOrAll: `Missing targets: provide package names or use --all

Examples:
  keg %[2]s foo bar      # %[1]s specific packages
  keg %[2]s --all        # %[1]s all packages listed in keg.yml`,

	AllWithRemoveNeedsForce: `--all with --remove requires --force

Usage:
  - Uninstall everything listed in keg.yml (keep manifest):
      keg delete --all
  - Uninstall everything and purge keg.yml (destructive):
      keg delete --all --remove --force

Reason:
  --remove deletes entries in keg.yml. Combined with --all it purges the manifest.
  --force is required to acknowledge this destructive operation.`,

	AllWithAddInvalid: `Invalid flag combination: cannot combine --all with --add

Usage:
  - Install all packages from keg.yml:
      keg install --all
  - Install a specific package and add it to keg.yml:
      keg install foo --add

Reason:
  --all targets everything from the manifest; --add mutates the manifest for specific targets.`,

	OptOrBinRequireAdd: `Invalid flag combination: --optional and --binary require --add

Usage:
  keg install foo --add --optional
  keg install foo --add --binary batcat

Reason:
  --optional and --binary only apply when adding the package to keg.yml.`,

	BinarySinglePackageOnly: `Invalid usage: --binary can only be used with a single package

Usage:
  keg install foo --add --binary batcat`,
}

func Msg(code Code, a ...any) string {
	msg := messages[code]
	if msg == "" {
		msg = string(code)
	}
	return fmt.Sprintf(msg, a...)
}
