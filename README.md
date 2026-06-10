# Terraform Provider for Firezone

A Terraform provider for managing Firezone VPN resources including actors, groups, sites, resources, policies, and authentication providers.

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Resources](#resources)
  - [Actors](#firezone_actor)
  - [Groups](#firezone_group)
  - [Group Memberships](#firezone_group_membership)
  - [Sites](#firezone_site)
  - [Resources](#firezone_resource)
  - [Policies](#firezone_policy)
  - [Auth Providers](#authentication-providers)
- [Examples](#examples)
- [Importing Existing Resources](#importing-existing-resources)
- [Troubleshooting](#troubleshooting)
- [Scripts](#scripts)
- [Development](#development)

## Quick Start

```bash
# Install the provider
./install.sh

# Create a Terraform configuration
cat > main.tf <<EOF
terraform {
  required_providers {
    firezone = {
      source  = "registry.terraform.io/example/firezone"
      version = "0.1.0"
    }
  }
}

provider "firezone" {
  api_url = "https://firezone-api.us-services.cardoaiapps.com"
  api_key = var.firezone_api_key
}

resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_actor" "john" {
  name  = "John Doe"
  email = "john.doe@example.com"
  type  = "account_user"
}
EOF

# Initialize and apply
terraform init
terraform apply
```

## Installation

### Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)

### Method 1: Automatic Install (Recommended)

```bash
git clone https://github.com/yourorg/terraform-provider-firezone.git
cd terraform-provider-firezone
./install.sh
```

### Method 2: Manual Install

```bash
# Build the provider
go build -o terraform-provider-firezone

# Install (macOS ARM64)
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/darwin_arm64
cp terraform-provider-firezone ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/darwin_arm64/

# Install (Linux AMD64)
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/linux_amd64
cp terraform-provider-firezone ~/.terraform.d/plugins/registry.terraform.io/example/firezone/0.1.0/linux_amd64/
```

### Method 3: Development Override

For active development, create `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/example/firezone" = "/absolute/path/to/terraform-provider-firezone"
  }
  direct {}
}
```

## Configuration

### Provider Block

```hcl
provider "firezone" {
  api_url = "https://firezone-api.us-services.cardoaiapps.com"
  api_key = var.firezone_api_key
}
```

### Environment Variables

```bash
export FIREZONE_API_URL="https://firezone-api.us-services.cardoaiapps.com"
export FIREZONE_API_KEY="your-api-key"
```

## Resources

### firezone_actor

Manages Firezone actors (users and service accounts).

**Actor Types:**
- `account_user` - Regular user
- `account_admin_user` - Admin user
- `service_account` - Service/automation account

**Arguments:**
- `name` (Required) - Actor name
- `email` (Required) - Actor email (must be unique)
- `type` (Required) - Actor type
- `allow_email_otp_sign_in` (Optional) - Allow email OTP authentication

**Computed Attributes:**
- `id` - Actor ID (UUID)
- `inserted_at` - Creation timestamp
- `updated_at` - Last update timestamp
- `last_seen_at` - Last activity timestamp

**Example:**

```hcl
# Regular user
resource "firezone_actor" "john" {
  name  = "John Doe"
  email = "john.doe@example.com"
  type  = "account_user"
}

# Admin user
resource "firezone_actor" "admin" {
  name  = "Admin User"
  email = "admin@example.com"
  type  = "account_admin_user"
}

# Service account for CI/CD
resource "firezone_actor" "cicd" {
  name  = "CI/CD Pipeline"
  email = "cicd@example.com"
  type  = "service_account"
}

# Multiple users with for_each
locals {
  team_members = {
    jane = {
      name  = "Jane Smith"
      email = "jane.smith@example.com"
    }
    bob = {
      name  = "Bob Johnson"
      email = "bob.johnson@example.com"
    }
  }
}

resource "firezone_actor" "team" {
  for_each = local.team_members
  
  name  = each.value.name
  email = each.value.email
  type  = "account_user"
}
```

### firezone_group

Manages Firezone groups.

**Arguments:**
- `name` (Required) - Group name

**Example:**

```hcl
resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_group" "devops" {
  name = "DevOps"
}
```

### firezone_group_membership

Manages group memberships by adding actors to groups.

**Arguments:**
- `group_id` (Required) - The group ID
- `actor_ids` (Required) - Set of actor IDs to add to the group

**Example:**

```hcl
# Create group and actors
resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_actor" "john" {
  name  = "John Doe"
  email = "john.doe@example.com"
  type  = "account_user"
}

resource "firezone_actor" "jane" {
  name  = "Jane Smith"
  email = "jane.smith@example.com"
  type  = "account_user"
}

# Add actors to group
resource "firezone_group_membership" "engineering_members" {
  group_id = firezone_group.engineering.id
  
  actor_ids = [
    firezone_actor.john.id,
    firezone_actor.jane.id,
  ]
}

# Dynamic membership
locals {
  team_members = {
    john = firezone_actor.john.id
    jane = firezone_actor.jane.id
  }
}

resource "firezone_group_membership" "dynamic_team" {
  group_id  = firezone_group.engineering.id
  actor_ids = values(local.team_members)
}
```

### firezone_site

Manages Firezone sites (VPN locations).

**Arguments:**
- `name` (Required) - Site name

**Example:**

```hcl
resource "firezone_site" "main" {
  name = "vpc-us-east"
}

resource "firezone_site" "backup" {
  name = "vpc-eu-west"
}
```

### firezone_resource

Manages network resources (IP addresses, CIDR blocks, or DNS names).

**Arguments:**
- `name` (Required) - Resource name
- `address` (Required) - Resource address
- `type` (Required) - Resource type: `ip`, `cidr`, or `dns`
- `site_id` (Required) - Site ID
- `address_description` (Optional) - Address description

**Example:**

```hcl
# IP resource
resource "firezone_resource" "database" {
  name                = "Production Database"
  address             = "10.0.0.10"
  address_description = "Main production database"
  type                = "ip"
  site_id             = firezone_site.main.id
}

# CIDR resource
resource "firezone_resource" "private_network" {
  name                = "Private Network"
  address             = "10.0.0.0/24"
  address_description = "Internal network"
  type                = "cidr"
  site_id             = firezone_site.main.id
}

# DNS resource
resource "firezone_resource" "internal_api" {
  name                = "Internal API"
  address             = "api.internal.example.com"
  address_description = "Internal API endpoint"
  type                = "dns"
  site_id             = firezone_site.main.id
}
```

### firezone_policy

Manages access policies connecting groups to resources.

**Arguments:**
- `group_id` (Required) - Group ID
- `resource_id` (Required) - Resource ID
- `description` (Optional) - Policy description

**Example:**

```hcl
resource "firezone_policy" "eng_db_access" {
  description = "Allow engineering team to access production database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.database.id
}
```

### Authentication Providers

#### firezone_google_auth_provider

Manages Google OAuth2 authentication.

**Setup Google OAuth2:**
1. Create OAuth 2.0 Client ID in Google Cloud Console
2. Add authorized redirect URI: `https://your-firezone-domain/auth/google/callback`
3. Note your Client ID and Client Secret

**Arguments:**
- `name` (Required) - Provider name
- `client_id` (Required) - Google OAuth2 Client ID
- `client_secret` (Required, Sensitive) - Google OAuth2 Client Secret
- `context` (Optional) - Where the provider can be used: `clients_and_portal`, `clients_only`, `portal_only` (default: `clients_and_portal`)
- `is_default` (Optional) - Whether this is the default provider (default: `false`)
- `is_disabled` (Optional) - Whether the provider is disabled (default: `false`)
- `client_session_lifetime_secs` (Optional) - Client session lifetime in seconds (default: `86400`)
- `portal_session_lifetime_secs` (Optional) - Portal session lifetime in seconds (default: `3600`)

**Example:**

```hcl
resource "firezone_google_auth_provider" "google" {
  name          = "Google SSO"
  client_id     = var.google_client_id
  client_secret = var.google_client_secret
  is_default    = true
  context       = "clients_and_portal"
  
  client_session_lifetime_secs = 86400  # 24 hours
  portal_session_lifetime_secs = 3600   # 1 hour
}
```

### firezone_oidc_auth_provider

Manages OIDC authentication provider (works with Keycloak, Auth0, Okta, Azure AD, and any OIDC-compliant provider).

**Arguments:**
- `name` (Required) - Provider name
- `client_id` (Required) - OAuth2 Client ID
- `client_secret` (Required, Sensitive) - OAuth2 Client Secret
- `discovery_document_uri` (Required) - OIDC Discovery Document URI (e.g., `https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration`)
- `scope` (Optional) - OAuth2 scope (default: `openid email profile`)
- `context` (Optional) - Where the provider can be used: `clients_and_portal`, `clients_only`, `portal_only` (default: `clients_and_portal`)
- `is_default` (Optional) - Whether this is the default provider (default: `false`)
- `is_disabled` (Optional) - Whether the provider is disabled (default: `false`)
- `client_session_lifetime_secs` (Optional) - Client session lifetime in seconds (default: `86400`)
- `portal_session_lifetime_secs` (Optional) - Portal session lifetime in seconds (default: `3600`)

**Examples:**

```hcl
# Keycloak
resource "firezone_oidc_auth_provider" "keycloak" {
  name                   = "Keycloak"
  client_id              = var.keycloak_client_id
  client_secret          = var.keycloak_client_secret
  discovery_document_uri = "https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration"
  scope                  = "openid email profile"
  is_default             = true
}

# Auth0
resource "firezone_oidc_auth_provider" "auth0" {
  name                   = "Auth0"
  client_id              = var.auth0_client_id
  client_secret          = var.auth0_client_secret
  discovery_document_uri = "https://your-tenant.auth0.com/.well-known/openid-configuration"
  context                = "clients_and_portal"
}

# Okta
resource "firezone_oidc_auth_provider" "okta" {
  name                   = "Okta"
  client_id              = var.okta_client_id
  client_secret          = var.okta_client_secret
  discovery_document_uri = "https://your-domain.okta.com/.well-known/openid-configuration"
}

# Multiple providers
resource "firezone_google_auth_provider" "google" {
  name       = "Google SSO"
  client_id  = var.google_client_id
  client_secret = var.google_client_secret
  is_default = true
}

resource "firezone_oidc_auth_provider" "contractors" {
  name                   = "Contractor Portal"
  client_id              = var.contractor_client_id
  client_secret          = var.contractor_client_secret
  discovery_document_uri = var.contractor_discovery_uri
  is_default             = false
  context                = "portal_only"
}
```

## Data Sources

### firezone_actor

Query existing actors.

```hcl
data "firezone_actor" "existing" {
  id = "actor-id-here"
}

output "actor_email" {
  value = data.firezone_actor.existing.email
}
```

### firezone_group

Query existing groups.

```hcl
data "firezone_group" "existing" {
  id = "group-id-here"
}
```

### firezone_google_auth_provider / firezone_oidc_auth_provider

Query existing auth providers.

```hcl
data "firezone_oidc_auth_provider" "existing" {
  id = "provider-id"
}

output "provider_status" {
  value = {
    name       = data.firezone_oidc_auth_provider.existing.name
    is_default = data.firezone_oidc_auth_provider.existing.is_default
  }
}
```

## Examples

### Complete Infrastructure Setup

```hcl
# 1. Create groups
resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_group" "devops" {
  name = "DevOps"
}

# 2. Create users
resource "firezone_actor" "john" {
  name  = "John Doe"
  email = "john@example.com"
  type  = "account_user"
}

resource "firezone_actor" "jane" {
  name  = "Jane Smith"
  email = "jane@example.com"
  type  = "account_admin_user"
}

# 3. Add users to groups
resource "firezone_group_membership" "eng_members" {
  group_id = firezone_group.engineering.id
  actor_ids = [firezone_actor.john.id]
}

resource "firezone_group_membership" "devops_members" {
  group_id = firezone_group.devops.id
  actor_ids = [firezone_actor.jane.id]
}

# 4. Create site
resource "firezone_site" "main" {
  name = "vpc-us-east"
}

# 5. Create resources
resource "firezone_resource" "database" {
  name                = "Production Database"
  address             = "10.0.0.10"
  address_description = "Main database"
  type                = "ip"
  site_id             = firezone_site.main.id
}

resource "firezone_resource" "app_servers" {
  name                = "Application Servers"
  address             = "10.0.1.0/24"
  address_description = "App server subnet"
  type                = "cidr"
  site_id             = firezone_site.main.id
}

# 6. Create policies
resource "firezone_policy" "eng_db_access" {
  description = "Engineering can access database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.database.id
}

resource "firezone_policy" "devops_all_access" {
  description = "DevOps can access all servers"
  group_id    = firezone_group.devops.id
  resource_id = firezone_resource.app_servers.id
}

# 7. Setup authentication
resource "firezone_google_auth_provider" "google" {
  name          = "Google SSO"
  client_id     = var.google_client_id
  client_secret = var.google_client_secret
  is_default    = true
}
```

See the [examples/](examples/) directory for more complete examples.

## Importing Existing Resources

### Getting Resource IDs

Use the Firezone API or data sources to find resource IDs:

```bash
export FIREZONE_API_KEY="your-api-key"
export FIREZONE_API_URL="https://firezone-api.us-services.cardoaiapps.com"

# List all actors
curl -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/actors"

# List all groups
curl -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/groups"

# List all auth providers
curl -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/oidc_auth_providers"
```

### Import Workflow

1. **Create the resource block** in your Terraform configuration
2. **Import the resource** using its ID
3. **Run terraform plan** to see what changes would be made
4. **Adjust your config** to match the existing state

**Example:**

```bash
# 1. Add to your .tf file:
# resource "firezone_actor" "john" {
#   name  = "John Doe"
#   email = "john@example.com"
#   type  = "account_user"
# }

# 2. Import
terraform import firezone_actor.john abc-123-def-456

# 3. Verify
terraform plan

# 4. Adjust config if needed
terraform show
```

### Import Commands Reference

```bash
# Actors
terraform import firezone_actor.name <actor-id>

# Groups
terraform import firezone_group.name <group-id>

# Group Memberships
terraform import firezone_group_membership.name <group-id>

# Sites
terraform import firezone_site.name <site-id>

# Resources
terraform import firezone_resource.name <resource-id>

# Policies
terraform import firezone_policy.name <policy-id>

# Auth Providers
terraform import firezone_google_auth_provider.name <provider-id>
terraform import firezone_oidc_auth_provider.name <provider-id>
```

## Troubleshooting

### 404 Not Found Error

**Symptom:** Getting HTML 404 page instead of JSON response.

**Cause:** API URL is pointing to the web UI instead of the API endpoint.

**Solution:**

1. Test your API endpoint:
```bash
curl -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "https://firezone-api.us-services.cardoaiapps.com/account"
```

2. Use the correct URL format:
```hcl
# ❌ Wrong - Web UI URL
api_url = "https://vpn.example.com"

# ✅ Correct - API URL
api_url = "https://firezone-api.us-services.cardoaiapps.com"
```

3. Verify no trailing slash or extra path:
```hcl
# ❌ Wrong
api_url = "https://firezone-api.us-services.cardoaiapps.com/"
api_url = "https://firezone-api.us-services.cardoaiapps.com/api"

# ✅ Correct
api_url = "https://firezone-api.us-services.cardoaiapps.com"
```

### 401 Unauthorized Error

**Solution:** Verify your API key is correct and has proper permissions.

```bash
# Test authentication
curl -v -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/account"
```

### 429 Rate Limit Error

**Solution:** The API has rate limits. Add delays between operations or reduce parallelism.

### Enable Debug Logging

```bash
export TF_LOG=DEBUG
terraform apply
```

Look for lines showing the actual URLs being called.

### Testing Script

Save as `test_api.sh`:

```bash
#!/bin/bash

if [ -z "$FIREZONE_API_URL" ] || [ -z "$FIREZONE_API_KEY" ]; then
  echo "Error: Set FIREZONE_API_URL and FIREZONE_API_KEY"
  exit 1
fi

echo "Testing Firezone API..."
echo "URL: $FIREZONE_API_URL"

# Test account endpoint
echo -e "\nTest: GET /account"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/account" | head -c 200

# Test list actors
echo -e "\n\nTest: GET /actors"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  -H "Authorization: Bearer $FIREZONE_API_KEY" \
  "$FIREZONE_API_URL/actors" | head -c 200
```

## Scripts

Both scripts live in [scripts/](scripts/) and require AWS SSO access via `cardoai-us-infra-mgmt` profile. Run them from the repo root.

### scripts/install.sh

Builds Go binaries for all platforms and uploads them to the private S3 mirror.

**Prerequisites:** `aws`, `jq`, `go` >= 1.21, a git tag (version is derived from `git describe --tags`).

```bash
# Tag first, then run
git tag v0.2.0
bash scripts/install.sh
```

What it does:
1. Assumes IAM role chain → lands in account `412381766327`
2. Builds `terraform-provider-firezone_v<VERSION>` for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` into `dist/`
3. Uploads each binary to `s3://cardoai-us-cw-firezone-provider-bucket/registry.terraform.io/cardoai/firezone/<version>/<os>_<arch>/`

### scripts/setup-terraform-mirror.sh

Downloads the provider binary for the current platform from S3 and configures a workspace-local `.terraformrc` so `terraform init` picks it up without hitting the public registry.

**Prerequisites:** `aws`, `jq`. Target Terraform workspace directory must already exist.

```bash
bash scripts/setup-terraform-mirror.sh
```

What it does:
1. Assumes the same IAM role chain
2. Detects current OS/arch
3. Syncs the matching binary from S3 into `<target-dir>/.terraform/plugins/`
4. Writes `<target-dir>/.terraformrc` pointing at the local mirror

After running, activate the config before `terraform init`:

```bash
cd cardoai-us/cardoai-us-infra/cardoai-us-common-wl-412381766327/firezone
export TF_CLI_CONFIG_FILE=$(pwd)/.terraformrc
terraform init
```

> **Note:** Add `.terraform/` and `.terraformrc` to `.gitignore` in the target workspace.

## Development

### Building from Source

```bash
go mod download
go build -o terraform-provider-firezone
```

### Running Tests

```bash
# Unit tests
go test ./...

# Acceptance tests (requires valid API credentials)
TF_ACC=1 FIREZONE_API_URL="https://your-api.com" \
  FIREZONE_API_KEY="your-key" \
  go test ./... -v
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

MIT
