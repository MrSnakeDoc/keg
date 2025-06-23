#!/bin/bash
set -euo pipefail

# ────────────── Config ───────────────────────────────────────────────────── #
REPO="MrSnakeDoc/keg"
BIN_NAME="keg"
INSTALL_DIR="${HOME}/.local/bin"
GH_API="https://api.github.com/repos/${REPO}/releases"

# ────────────── Colors & banner ────────────────────────────── #
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo -e "   _  __          "
echo -e "  | |/ /___  ____ "
echo -e "  | ' // _ \/ _  |"
echo -e "  | . \  __/ (_| |"
echo -e "  |_|\_\___|\__, |"
echo -e "            |___/ "
echo -e "${NC}"
echo -e "Installer for ${BLUE}keg${NC} - https://github.com/$REPO"
echo -e ""


curlNeeded() {
  if ! command -v curl >/dev/null 2>&1; then
    printf "%bError: curl is required to run this script.%b\n" "$RED" "$NC"
    printf "%bPlease install curl and try again.%b\n" "$YELLOW" "$NC"
    printf "%bFor example, on Debian/Ubuntu: sudo apt install curl%b\n" "$YELLOW" "$NC"
    printf "%bOr on arch-based systems: sudo pacman -S curl%b\n" "$YELLOW" "$NC"
    printf "%bOr on fedora: sudo dnf install curl%b\n" "$YELLOW" "$NC"
    exit 1
  fi
}


# ────────────── Pre-conditions ───────────────────────────────────────────── #
curlNeeded

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[[ $OS == linux ]] || { printf "%bError: keg only supports Linux systems.%b\n" "$RED" "$NC"; exit 1; }
case $ARCH in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) printf "%bError: unsupported arch: %s%b\n" "$RED" "$ARCH" "$NC"; exit 1 ;;
esac

# ────────────── Version parsing ─────────────────────────────────────── #
if [[ -z ${VERSION:-} ]]; then
  printf "%bVersion not specified, fetching latest...%b\n" "$YELLOW" "$NC"
  VERSION=$(curl -fsSL "${GH_API}/latest" | grep -oP '"tag_name":\s*"\Kv?[^"]+')
  [[ -n $VERSION ]] || { printf "%bError: could not determine latest version.%b\n" "$RED" "$NC"; exit 1; }
fi
VERSION=${VERSION#v}
printf "%bInstalling keg version %s...%b\n" "$BLUE" "$VERSION" "$NC"

ASSET="keg_${VERSION}_${OS}_${ARCH}"
DL_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"

# ────────────── Download + SHA-256 check ─────────────────────── #
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

printf "%bDownloading %s...%b\n" "$BLUE" "$DL_URL" "$NC"
curl -fL# -o "${TMP_DIR}/${ASSET}" "$DL_URL"

printf "%bVerifying checksum...%b\n" "$BLUE" "$NC"
EXPECTED=$(curl -fsSL "$CHECKSUM_URL" | grep " ${ASSET}$" | awk '{print $1}')
[[ -n $EXPECTED ]] || { printf "%bError: checksum not found for %s%b\n" "$RED" "$ASSET" "$NC"; exit 1; }
echo -e "${EXPECTED}  ${TMP_DIR}/${ASSET}" | sha256sum -c -

# ──────────────  Atomic installation with rollback ──────────────────────── #
mkdir -p "$INSTALL_DIR"
TARGET="${INSTALL_DIR}/${BIN_NAME}"

rollback() {
  if [[ -f ${TARGET}.old ]]; then
    mv -f "${TARGET}.old" "$TARGET"
    printf "%bRollback completed.%b\n" "$YELLOW" "$NC"
  fi
}
trap rollback ERR

if [[ -x $TARGET ]]; then
  CUR_VER=$("$TARGET" version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo -e unknown)
  printf "%bUpdating keg from version %s to %s...%b\n" "$YELLOW" "$CUR_VER" "$VERSION" "$NC"
  mv -f "$TARGET" "${TARGET}.old"
fi

install -m 755 "${TMP_DIR}/${ASSET}" "$TARGET"
trap - ERR   

printf "%bkeg v%s has been installed to %s%b\n" "$GREEN" "$VERSION" "$TARGET" "$NC"

if ! command -v keg >/dev/null 2>&1; then
  printf "%bWARNING: %s is not in your PATH.%b\n" "$YELLOW" "$INSTALL_DIR" "$NC"
  echo -e "Add this to your shell profile:"
  echo -e "  export PATH=\"\$PATH:${INSTALL_DIR}\""
fi

rm -f "${TARGET}.old" 2>/dev/null || true
printf "%bThank you for installing keg! Run 'keg help' to get started.%b\n" "$GREEN" "$NC"