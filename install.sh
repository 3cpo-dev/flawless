#!/bin/sh
# flawless installer — downloads the latest release binary for your
# platform into ~/.local/bin (or /usr/local/bin when run as root).
# That is all it does: no daemon, no services, no config written.
set -eu

REPO="3cpo-dev/flawless"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "flawless: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin | linux) ;;
  *) echo "flawless: use the Windows binary from https://github.com/$REPO/releases" >&2; exit 1 ;;
esac

if [ "$(id -u)" = 0 ]; then
  bindir=/usr/local/bin
else
  bindir="$HOME/.local/bin"
  mkdir -p "$bindir"
fi

url="https://github.com/$REPO/releases/latest/download/flawless_${os}_${arch}"
echo "downloading $url"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
curl -fsSL -o "$tmp" "$url"
chmod +x "$tmp"
mv "$tmp" "$bindir/flawless"
trap - EXIT

echo "installed $bindir/flawless ($("$bindir/flawless" version))"
case ":$PATH:" in
  *":$bindir:"*) ;;
  *) echo "note: $bindir is not on your PATH — add it to your shell profile" ;;
esac
echo "next: cd into a repo and run: flawless doctor"
