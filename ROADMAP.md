# KEG CLI Roadmap

## v0.1 - First public usable version

- [x] Init project
- [x] Load and save config
- [x] List packages
- [x] Install packages
- [x] Uninstall packages
- [x] Update packages
- [x] Check update for installed packages
- [x] Delete packages
- [x] Handle missing config cleanly
- [x] Handle missing packages cleanly
- [x] Handle missing package manager cleanly
- [x] Install automatically zsh shell
- [x] Install automatically brew package manager
- [x] Install automatically all packages from config with brew
- [x] Runner for mocking commands
- [x] Add command to add packages to config
- [x] Remove command to remove packages from config
- [x] Implement structured PackageHandlerOptions for better flexibility
- [x] Support for binary name different from package name
- [x] Support for optional packages
- [x] Improved cache handling for brew outdated packages
- [x] Validation functions for package operations
- [x] Better error handling with clear user messages
- [x] File utilities for consistent file operations
- [x] Version comparison utilities for semantic versioning
- [x] Add command validation to prevent incorrect usage
- [x] Command execution abstraction with runners
- [x] Consistent config file handling with SaveConfig utility
- [x] Polish logging
- [x] Autoupdate of keg cli

## v0.2 - Advanced usability

- [ ] Multi-profiles support (~/.config/keg/profiles/)
- [ ] Custom config locations
- [ ] Basic plugin system
- [ ] Global `--dry-run / -n` flag
- [ ] Centralized AppFlags struct for global CLI flags
- [ ] Propagate dry-run flag to all commands
- [ ] Refactor command logic to respect dry-run behavior
- [ ] Optional: Introduce ConfigSaver interface for dry-run vs. real saves

## v0.3 - Major improvements

- [ ] Caching downloaded packages
- [ ] Hooks system (pre/post install scripts)
- [ ] Remote registries support
- [ ] Package search
