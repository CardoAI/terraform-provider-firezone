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
    --role-session-name "terraform-provider-upload")

export AWS_ACCESS_KEY_ID=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.AccessKeyId)
export AWS_SECRET_ACCESS_KEY=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.SecretAccessKey)
export AWS_SESSION_TOKEN=$(echo ${TARGET_CREDENTIALS} | jq -r .Credentials.SessionToken)

echo "✅ Successfully assumed role in account 412381766327"

# Configuration
PROVIDER_NAME="firezone"
NAMESPACE="cardoai"
S3_BUCKET="${NAMESPACE}-us-cw-firezone-provider-bucket"
AWS_REGION="${AWS_REGION:-us-east-2}"

# Get version from git tag (latest tag)
VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
if [ -z "$VERSION" ]; then
    echo "Error: No git tags found. Please create a tag first."
    exit 1
fi

echo "Building version: ${VERSION}"

# Create build directory
BUILD_DIR="../dist"
mkdir -p "${BUILD_DIR}"

# Architectures and OS to build for
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    
    echo "Building for ${OS}/${ARCH}..."
    
    OUTPUT_NAME="terraform-provider-${PROVIDER_NAME}_v${VERSION}"
    export FIREZONE_RATE_LIMIT_RPS="200"
    export FIREZONE_RATE_LIMIT_BURST="500"

    GOOS="${OS}" GOARCH="${ARCH}" go build -o "${BUILD_DIR}/${OUTPUT_NAME}_${OS}_${ARCH}" ../
    
    echo "✅ Built ${OUTPUT_NAME}_${OS}_${ARCH}"
done

# Upload to S3
echo "Uploading to S3 bucket: ${S3_BUCKET}"

for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    OUTPUT_NAME="terraform-provider-${PROVIDER_NAME}_v${VERSION}_${OS}_${ARCH}"
    # Terraform filesystem mirror expects: registry.terraform.io/namespace/provider/version/os_arch/
    S3_PATH="s3://${S3_BUCKET}/registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS}_${ARCH}/terraform-provider-${PROVIDER_NAME}_v${VERSION}"
    
    aws s3 cp "${BUILD_DIR}/${OUTPUT_NAME}" "${S3_PATH}" --region "${AWS_REGION}"
    echo "✅ Uploaded to ${S3_PATH}"
done

echo ""
echo "✅ All builds completed and uploaded successfully!"
echo ""
echo "Configure Terraform CLI to use the mirror:"
echo ""
echo "provider_installation {"
echo "  filesystem_mirror {"
echo "    path    = \"s3://${S3_BUCKET}\""
echo "    include = [\"registry.terraform.io/${NAMESPACE}/${PROVIDER_NAME}\"]"
echo "  }"
echo "}"
