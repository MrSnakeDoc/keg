![Go](https://img.shields.io/badge/go-1.24%2B-blue)
![goreleaser](https://github.com/MrSnakeDoc/keg/actions/workflows/release.yml/badge.svg)

# Keg CLI

A modern, opinionated CLI to automate and manage your Linux development environment with style. 

---

Table of Contents
- [Keg CLI](#keg-cli)
  - [🚀 Features](#-features)
  - [🖥️ Prerequisites](#️-prerequisites)
  - [📦 Installation](#-installation)
    - [From Bash Script](#from-bash-script)
    - [From Source](#from-source)
  - [⚙️ Configuration](#️-configuration)
  - [🛠️ Usage](#️-usage)
    - [Bootstrap your shell](#bootstrap-your-shell)
    - [Deploy your environment](#deploy-your-environment)
    - [Package management](#package-management)
    - [Update Keg itself](#update-keg-itself)
    - [Global options](#global-options)
  - [🧪 Testing \& Development](#-testing--development)
  - [📝 Roadmap](#-roadmap)
  - [📄 License](#-license)
  - [💡 Tips](#-tips)
---

## 🚀 Features

- 🚀 ZSH auto-installation and configuration
- 🍺 Homebrew auto-installation and management
- 📋 Package management (install, upgrade, remove, list, add, delete, remove)
- ⚡ Optional package support
- 🗄️ Centralized config and state management
- 🔄 Automatic CLI self-update (via GitHub Releases)
- ❌ Robust error handling and clear user feedback
- 🧪 Mockable runners and HTTP clients for easy testing

---

## 🖥️ Prerequisites

- Linux (Ubuntu, Fedora, Arch tested)
- Go 1.21+ (for building from source)
- Homebrew (auto-installed if missing)

---

## 📦 Installation

### From Bash Script

```bash
curl -fsSL https://raw.githubusercontent.com/MrSnakeDoc/keg/main/scripts/install.sh | bash -
```

### From Source

```bash
git clone https://github.com/MrSnakeDoc/keg.git
cd keg
make build
```

---

## ⚙️ Configuration

- Run `keg init` to create a `keg.yml` in your current directory and initialize global config in `~/.config/keg`.
- All package operations are based on this config file.
- State and update info are stored in `~/.local/state/keg/update-check.json`.

Example `keg.yml`:

```yaml
packages:
  - command: eza
  - command: bat
  - command: lazygit
    optional: true
```

---

## 🛠️ Usage

### Bootstrap your shell

```bash
keg bootstrap
```
- Installs ZSH if missing
- Sets ZSH as default shell

### Deploy your environment

```bash
keg deploy
```
- Installs Homebrew if missing
- Installs all packages from config

### Package management

```bash
keg install [packages...]      # Install packages (default: all non-optional)
keg install --all              # Install all packages (including optional)
keg add bat                    # Add package to config
git add --optional lazygit     # Add optional package to config
keg list                       # List all packages and their status
keg upgrade [packages...]      # Upgrade packages (default: all)
keg upgrade --check            # Check for available upgrades only
keg delete [packages...]       # Uninstall packages
keg delete --all               # Uninstall all packages
keg remove [packages...]       # Remove packages from config only
```

### Update Keg itself

```bash
keg update
```
- Checks for new version on GitHub
- Downloads and replaces the binary if needed
- Verifies SHA256 checksum

### Global options

```bash
keg --version                  # Show CLI version
keg -f config.yml              # Use a specific config file
keg --no-update-check          # Skip update check (for scripting)
```

---

## 🧪 Testing & Development

- All runners and HTTP clients are mockable for robust testing
- Use `make build` to build the CLI
- Use `make comp` to generate ZSH completions
- Tests use temp dirs and fake clients to avoid side effects

---

## 📝 Roadmap

See [ROADMAP.md](./ROADMAP.md) for planned features and progress.

---

## 📄 License

MIT License. See [LICENSE](./LICENSE) for details.

---

## 💡 Tips

- Keg is Linux-only by design. Mac support is possible but not tested. Windows users: WSL is your friend.
- All config/state is local and human-readable. No cloud, no telemetry, no bullshit.
- If you break it, you get to keep both pieces. PRs welcome!
