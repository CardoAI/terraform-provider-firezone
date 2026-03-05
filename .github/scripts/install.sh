#!/bin/bash
set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
esac

# Map OS names
case $OS in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    mingw*|msys*|cygwin*)
        OS="windows"
        ;;
esac

# Get version from git tag (latest tag)
VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
if [ -z "$VERSION" ]; then
    echo "Error: No git tags found. Please create a tag first."
    exit 1
fi

PROVIDER_NAME="firezone"
NAMESPACE="cardoai"
GITHUB_REPO="cardoai/firezone-terraform-provider"

# Create plugin directory
PLUGIN_DIR="${HOME}/.terraform.d/plugins/registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS}_${ARCH}"
echo "Installing to ${PLUGIN_DIR}..."
mkdir -p "${PLUGIN_DIR}"

# Download from GitHub releases
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/firezone-terraform-provider_${VERSION}_${OS}_${ARCH}.zip"
echo "Downloading from ${DOWNLOAD_URL}..."

curl -L -o /tmp/provider.zip "${DOWNLOAD_URL}"

# Extract binary
unzip -o /tmp/provider.zip -d /tmp/provider_extract
cp /tmp/provider_extract/firezone-terraform-provider_v${VERSION}* "${PLUGIN_DIR}/terraform-provider-${PROVIDER_NAME}"
chmod +x "${PLUGIN_DIR}/terraform-provider-${PROVIDER_NAME}"

# Cleanup
rm -rf /tmp/provider.zip /tmp/provider_extract

echo "✅ Provider installed successfully!"
echo ""
echo "Use in your Terraform configuration:"
echo ""
echo "terraform {"
echo "  required_providers {"
echo "    ${PROVIDER_NAME} = {"
echo "      source  = \"registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}\""
echo "      version = \"${VERSION}\""
echo "    }"
echo "  }"
echo "}"
