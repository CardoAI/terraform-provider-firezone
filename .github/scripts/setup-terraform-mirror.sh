#!/bin/bash
set -e

# AWS Configuration
export AWS_PROFILE="cardoai-us-infra-mgmt"
# if ! aws sts get-caller-identity --profile ${AWS_PROFILE} > /dev/null 2>&1; then
#   aws sso login --sso-session cardoai
# fi

# First assume the intermediate role in infra-mgmt account
INTERMEDIATE_ROLE_ARN="arn:aws:iam::221082183447:role/cardoai-us-im-iam-role-terraform-admin"
echo "Assuming intermediate role: ${INTERMEDIATE_ROLE_ARN}"

INTERMEDIATE_CREDENTIALS=$(aws sts assume-role \
    --role-arn "${INTERMEDIATE_ROLE_ARN}" \
    --role-session-name "terraform-intermediate" \
    --profile "${AWS_PROFILE}")

export AWS_ACCESS_KEY_ID=$(echo ${INTERMEDIATE_CREDENTIALS} | jq -r .Credentials.AccessKeyId)
export AWS_SECRET_ACCESS_KEY=$(echo ${INTERMEDIATE_CREDENTIALS} | jq -r .Credentials.SecretAccessKey)
export AWS_SESSION_TOKEN=$(echo ${INTERMEDIATE_CREDENTIALS} | jq -r .Credentials.SessionToken)

# Now assume the target role in common-wl account (412381766327)
TARGET_ROLE_ARN="arn:aws:iam::412381766327:role/cardoai-us-cw-iam-role-terraform-admin"
echo "Assuming target role: ${TARGET_ROLE_ARN}"

TARGET_CREDENTIALS=$(aws sts assume-role \
    --role-arn "${TARGET_ROLE_ARN}" \
    --role-session-name "terraform-mirror-sync")

export AWS_ACCESS_KEY_ID=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.AccessKeyId)
export AWS_SECRET_ACCESS_KEY=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.SecretAccessKey)
export AWS_SESSION_TOKEN=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.SessionToken)

echo "✅ Successfully assumed role in account 412381766327"

# Configuration
PROVIDER_NAME="firezone"
NAMESPACE="cardoai"
S3_BUCKET="cardoai-us-cw-firezone-provider-bucket"
AWS_REGION="us-east-2"
TARGET_DIR="./cardoai-us/cardoai-us-infra/cardoai-us-common-wl-412381766327/firezone"
MIRROR_PATH="${TARGET_DIR}/.terraform/plugins"
TERRAFORMRC="${TARGET_DIR}/.terraformrc"

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
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
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
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Detected platform: ${OS}_${ARCH}"
echo "Setting up Terraform workspace-level mirror..."
echo "Target directory: ${TARGET_DIR}"

# Create target directory if it doesn't exist
if [ ! -d "${TARGET_DIR}" ]; then
    echo "Error: Target directory ${TARGET_DIR} does not exist"
    exit 1
fi

# Create mirror directory in target location
mkdir -p "${MIRROR_PATH}/registry.terraform.io"

# Sync providers from S3 for current platform only
echo "Syncing providers from S3 for ${OS}_${ARCH}..."
aws s3 sync \
    "s3://${S3_BUCKET}/registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}" \
    "${MIRROR_PATH}/registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}" \
    --exclude "*" \
    --include "*/${OS}_${ARCH}/*" \
    --region "${AWS_REGION}"

# Make provider binaries executable
echo "Setting executable permissions on provider binaries..."
find "${MIRROR_PATH}/registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}" -type f -name "terraform-provider-*" -exec chmod +x {} \;

echo "✅ Providers synced to ${MIRROR_PATH}"

# Backup existing .terraformrc if it exists
if [ -f "${TERRAFORMRC}" ]; then
    BACKUP="${TERRAFORMRC}.backup.$(date +%Y%m%d_%H%M%S)"
    echo "Backing up existing .terraformrc to ${BACKUP}"
    cp "${TERRAFORMRC}" "${BACKUP}"
fi

# Create or update .terraformrc
cat > "${TERRAFORMRC}" << EOF
provider_installation {
  filesystem_mirror {
    path    = "${MIRROR_PATH}"
    include = ["registry.terraform.io/${NAMESPACE}/*"]
  }
  direct {
    exclude = ["registry.terraform.io/${NAMESPACE}/*"]
  }
}
EOF

echo "✅ Created ${TERRAFORMRC}"
echo ""
echo "Terraform is now configured to use the workspace mirror for ${NAMESPACE} providers."
echo "Files created in: ${TARGET_DIR}"
echo ""
echo "To use this configuration, navigate to the directory and run:"
echo ""
echo "  cd ${TARGET_DIR}"
echo "  export TF_CLI_CONFIG_FILE=\$(pwd)/.terraformrc"
echo "  terraform init"
echo ""
echo "In your Terraform configuration, use:"
echo ""
echo "terraform {"
echo "  required_providers {"
echo "    ${PROVIDER_NAME} = {"
echo "      source  = \"registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}\""
echo "      version = \"~> 0.1\""
echo "    }"
echo "  }"
echo "}"
echo ""
echo "To update providers, run this script again to sync from S3."
echo ""
echo "Note: Add .terraform/ and .terraformrc to .gitignore if not already present."
