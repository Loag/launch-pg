#!/bin/sh
# Install launch-pg from GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/Loag/launch-pg/master/install.sh | sh
#
# Environment:
#   VERSION      release tag to install (default: latest), e.g. v0.1.0
#   INSTALL_DIR  where to put the binary (default: /usr/local/bin if
#                writable, otherwise ~/.local/bin)
set -eu

REPO="Loag/launch-pg"
BINARY="launch-pg"

say() { printf '%s\n' "$*" >&2; }
fail() { say "error: $*"; exit 1; }

# fetch URL [OUTPUT] — download with curl or wget; stdout when no OUTPUT.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		if [ $# -eq 2 ]; then curl -fsSL -o "$2" "$1"; else curl -fsSL "$1"; fi
	elif command -v wget >/dev/null 2>&1; then
		if [ $# -eq 2 ]; then wget -qO "$2" "$1"; else wget -qO- "$1"; fi
	else
		fail "curl or wget is required"
	fi
}

detect_platform() {
	case "$(uname -s)" in
		Linux) OS=linux ;;
		Darwin) OS=darwin ;;
		*) fail "unsupported OS $(uname -s); download a release manually from https://github.com/$REPO/releases" ;;
	esac
	case "$(uname -m)" in
		x86_64 | amd64) ARCH=amd64 ;;
		arm64 | aarch64) ARCH=arm64 ;;
		*) fail "unsupported architecture $(uname -m)" ;;
	esac
}

resolve_version() {
	if [ -n "${VERSION:-}" ]; then
		return
	fi
	VERSION=$(fetch "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)
	[ -n "$VERSION" ] || fail "could not find the latest release of $REPO"
}

resolve_install_dir() {
	if [ -n "${INSTALL_DIR:-}" ]; then
		return
	fi
	if [ -w /usr/local/bin ]; then
		INSTALL_DIR=/usr/local/bin
	else
		INSTALL_DIR="$HOME/.local/bin"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

main() {
	detect_platform
	resolve_version
	resolve_install_dir

	# Archive names come from .goreleaser.yaml: the version has no leading "v".
	archive="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
	base="https://github.com/$REPO/releases/download/$VERSION"

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	say "Downloading $BINARY $VERSION ($OS/$ARCH)…"
	fetch "$base/$archive" "$tmp/$archive" || fail "download failed: $base/$archive"
	fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "could not download checksums.txt"

	expected=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
	[ -n "$expected" ] || fail "$archive is not listed in checksums.txt"
	[ "$(sha256 "$tmp/$archive")" = "$expected" ] || fail "checksum mismatch for $archive"

	tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY"
	mkdir -p "$INSTALL_DIR"
	install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY" 2>/dev/null ||
		fail "cannot write to $INSTALL_DIR; set INSTALL_DIR or run with sudo"

	say "Installed $INSTALL_DIR/$BINARY ($VERSION)"
	case ":$PATH:" in
		*":$INSTALL_DIR:"*) ;;
		*) say "Note: $INSTALL_DIR is not on your PATH. Add it, e.g.: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
	esac
}

main "$@"
