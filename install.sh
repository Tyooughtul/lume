#!/bin/bash

# Lume - One-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Tyooughtul/lume/main/install.sh | bash

set -e

REPO="Tyooughtul/lume"
BINARY_NAME="lume"
INSTALL_DIR="/usr/local/bin"

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 清理临时文件
cleanup() {
    rm -f "$TMP_FILE" 2>/dev/null
}
trap cleanup EXIT

# 检测架构
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64)       echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)            echo "unknown" ;;
    esac
}

# 检测最新版本
get_latest_version() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    echo ""
    echo "[*] Installing Lume..."
    echo ""

    # 检测操作系统
    if [ "$(uname -s)" != "Darwin" ]; then
        echo -e "${RED}[X] Lume only supports macOS${NC}"
        exit 1
    fi

    ARCH=$(detect_arch)
    if [ "$ARCH" = "unknown" ]; then
        echo -e "${RED}[X] Unsupported architecture: $(uname -m)${NC}"
        exit 1
    fi

    echo "→ Detected: macOS $ARCH"

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        echo -e "${RED}[X] Failed to fetch latest version${NC}"
        echo "  Please check your internet connection"
        exit 1
    fi

    echo "→ Latest version: $VERSION"

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/lume-darwin-${ARCH}"
    TMP_FILE=$(mktemp /tmp/lume.XXXXXX)

    echo "→ Downloading..."
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE" 2>/dev/null; then
        echo -e "${RED}[X] Download failed${NC}"
        echo "  URL: $DOWNLOAD_URL"
        exit 1
    fi

    chmod +x "$TMP_FILE"

    echo "→ Installing to $INSTALL_DIR..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        echo "  (sudo password required)"
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi

    echo ""
    echo -e "${GREEN}[OK] Lume installed successfully!${NC}"
    echo ""
    echo "  Run 'lume' to start cleaning"
    echo "  Run 'lume -help' for more options"
    echo ""
}

main "$@"
