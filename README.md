![Go](https://img.shields.io/badge/go-1.24%2B-blue)
![goreleaser](https://github.com/MrSnakeDoc/keg/actions/workflows/release.yml/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/MrSnakeDoc/keg)
![License](https://img.shields.io/badge/license-MIT-green)

# Keg CLI

A modern, opinionated CLI to automate and manage your Linux development environment with style.
Reproducible, idempotent, and fast.

---

## Why Keg Exists?

- Reinstalling Linux machines over and over wastes time. Instead of another brittle Bash script, I wanted an idempotent, reproducible, portable CLI: one `keg deploy` and my dev environment is back on Ubuntu, Fedora, or Arch.
- Keg does not replace Homebrew or Ansible; it’s a small, fast layer focused on developer experience—centralized config, a safe self-update path, and a clean, testable Go codebase.

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
- 📋 Package management (install, upgrade, delete, list, add, remove)
- ⚡ Optional packages
- 🗄️ Centralized config and state management
- 🔄 Automatic self-update (via GitHub Releases, with SHA256 verification)
- ❌ Robust error handling and clear user feedback
- 🧪 Mockable runners and HTTP clients for easy testing

---

## 🖥️ Prerequisites

- Linux (tested on Ubuntu, Fedora, Arch)
- Go 1.24+ (to build from source)
- Homebrew (auto-installed if missing)

---

## 📦 Installation

### From Bash Script

```bash
curl -fsSL https://raw.githubusercontent.com/MrSnakeDoc/keg/main/scripts/install.sh | bash -
```

 * Security & integrity

    - The installer fetches artifacts from official GitHub Releases only.
    - Each release publishes checksums.txt (and optionally a signature).
    - The script verifies the downloaded artifact against checksums.txt.
    - No third-party mirrors and no telemetry.

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
  - command: ripgrep
    binary: rg
    optional: true
```

---

## 🛠️ Usage

| Command                                  | What it does                                 |
| ---------------------------------------- | -------------------------------------------- |
| `keg bootstrap`                          | Install ZSH if missing and set it as default |
| `keg deploy`                             | Install Homebrew if needed + all packages    |
| `keg install [pkgs...]`                  | Install packages (default: all non-optional) |
| `keg install --all`                      | Install all packages (including optional)    |
| `keg add bat`                            | Add package to `keg.yml`                     |
| `keg add --optional ripgrep --binary rg` | Add optional package with custom binary      |
| `keg list`                               | List packages and their status               |
| `keg upgrade [pkgs...]`                  | Upgrade packages (default: all)              |
| `keg upgrade --check` or `-c`            | Only check for available upgrades            |
| `keg delete [pkgs...]`                   | Uninstall packages from the system           |
| `keg delete --all`                       | Uninstall all packages                       |
| `keg remove [pkgs...]`                   | Remove packages from config only             |
| `keg --version`                          | Show CLI version                             |
| `keg --no-update-check`                  | Skip update check (for scripting)            |


## 🔄 Update Keg itself

Keg provides a safe self-update mechanism:

* Every 24 hours it checks GitHub for a new version and notifies you.
* You can manually trigger a check with `keg update --check` (or `-c`).
* To update to the latest release, run `keg update` (SHA256 verified).

---

### Global options

```bash
keg --version               # Show CLI version
keg --no-update-check       # Skip update check (for scripting)
```

---

## 🧪 Testing & Development

* Runners and HTTP clients are mockable for robust tests.
* `make build` to build the CLI.
* `make comp` to generate ZSH completions.
* Tests use temp dirs and fake clients to avoid side effects.

---

## 🧱 Architecture

**High level:**

* **CLI / Commands**: thin cobra-style commands that delegate to services.
* **Planner**: computes idempotent actions (install/upgrade/delete) from `keg.yml`.
* **Runner**: executes actions via a small interface (`Exec(ctx, name, args...)`), easily mockable.
* **Providers**: distro-specific package providers (brew/apt/dnf/pacman).
* **Updater**: checks GitHub Releases, verifies SHA256, performs atomic binary replacement.
* **Config & State**: XDG paths; human-readable config; no telemetry.

---

## 📝 Roadmap

See [ROADMAP.md](./ROADMAP.md).

---

## 📄 License

MIT License — see [LICENSE](./LICENSE).

---

## 💡 Tips

* Linux-only by design. macOS might work via Linuxbrew but is not tested. Windows: use WSL. (Mostly because I don't own a Mac and I am not willing to pay an indecent amount of money for a device that I will use only to test Keg. No plans to support Windows natively because Windows is a spyware not an operating system.)
* Local, human-readable config/state. No cloud, no telemetry, no nonsense.
* If you break it, you get to keep both pieces. PRs welcome!