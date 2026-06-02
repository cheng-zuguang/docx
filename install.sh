#!/usr/bin/env sh
set -eu

DEFAULT_REPO_URL="https://github.com/cheng-zuguang/docx"
REPO="${DOCX_REPO:-cheng-zuguang/docx}"
VERSION="${DOCX_VERSION:-latest}"
INSTALL_DIR="${DOCX_INSTALL_DIR:-/usr/local/bin}"
BIN_NAME="${DOCX_BIN_NAME:-docx}"

uname_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
uname_arch="$(uname -m)"

case "$uname_os" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  *) echo "Unsupported OS: $uname_os" >&2; exit 1 ;;
esac

case "$uname_arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported architecture: $uname_arch" >&2; exit 1 ;;
esac

asset="docx_${os}_${arch}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
fi

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

echo "Downloading ${url}"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_dir/$asset"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_dir/$asset" "$url"
else
  echo "curl or wget is required" >&2
  exit 1
fi

tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
chmod +x "$tmp_dir/docx"
mkdir -p "$INSTALL_DIR"
mv "$tmp_dir/docx" "$INSTALL_DIR/$BIN_NAME"

echo "Installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
"$INSTALL_DIR/$BIN_NAME" --help >/dev/null
