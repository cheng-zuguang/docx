#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/npm/bin-runtime"
DIST_DIR="$ROOT_DIR/.dist/local"
BIN_NAME="${DOCX_BIN_NAME:-docx}"

case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
  mingw*|msys*|cygwin*) EXE_NAME="docx.exe" ;;
  *) EXE_NAME="docx" ;;
esac

rm -rf "$DIST_DIR"
mkdir -p "$RUNTIME_DIR" "$DIST_DIR"

echo "Building local docx binary from $ROOT_DIR"
export GOCACHE="$DIST_DIR/go-build-cache"
export GOPATH="$DIST_DIR/go"
mkdir -p "$GOCACHE" "$GOPATH"
go build -o "$RUNTIME_DIR/$EXE_NAME" ./cmd/docx
chmod +x "$RUNTIME_DIR/$EXE_NAME" 2>/dev/null || true

echo "Packing npm package with local runtime binary"
(
  cd "$ROOT_DIR"
  npm pack --pack-destination "$DIST_DIR" >/dev/null
)

PACKAGE_TGZ="$(find "$DIST_DIR" -name '*.tgz' -type f | head -n 1)"
if [ -z "$PACKAGE_TGZ" ]; then
  echo "No npm package was produced in $DIST_DIR" >&2
  exit 1
fi

install_args="-g"
if [ "${DOCX_LOCAL_PREFIX:-}" != "" ]; then
  install_args="$install_args --prefix $DOCX_LOCAL_PREFIX"
fi

echo "Installing local package: $PACKAGE_TGZ"
# The package already contains npm/bin-runtime/docx, so postinstall must not
# download a release asset. This script is intentionally for local debugging.
DOCX_SKIP_DOWNLOAD=1 npm install $install_args "$PACKAGE_TGZ"

if [ "${DOCX_LOCAL_PREFIX:-}" != "" ]; then
  INSTALLED_BIN="$DOCX_LOCAL_PREFIX/bin/$BIN_NAME"
else
  INSTALLED_BIN="$(command -v "$BIN_NAME" || true)"
fi

echo "Installed local debug package."
if [ "$INSTALLED_BIN" != "" ] && [ -x "$INSTALLED_BIN" ]; then
  echo "Binary: $INSTALLED_BIN"
  "$INSTALLED_BIN" --help >/dev/null
else
  echo "Add the npm global bin directory to PATH if '$BIN_NAME' is not found." >&2
fi
