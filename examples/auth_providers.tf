# Example: Setting up auth providers

# Create a Google Auth Provider
resource "firezone_google_auth_provider" "google" {
  name          = "Google SSO"
  client_id     = var.google_client_id
  client_secret = var.google_client_secret
  
  # Optional settings
  context                       = "clients_and_portal"
  is_default                    = true
  is_disabled                   = false
  client_session_lifetime_secs  = 86400  # 24 hours
  portal_session_lifetime_secs  = 3600   # 1 hour
}

# Create an OIDC Auth Provider (e.g., for Keycloak, Auth0, etc.)
resource "firezone_oidc_auth_provider" "keycloak" {
  name                    = "Keycloak SSO"
  client_id               = var.keycloak_client_id
  client_secret           = var.keycloak_client_secret
  discovery_document_uri  = "https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration"
  
  # Optional settings
  scope                         = "openid email profile"
  context                       = "clients_and_portal"
  is_default                    = false
  is_disabled                   = false
  client_session_lifetime_secs  = 86400
  portal_session_lifetime_secs  = 3600
}

# Import an existing OIDC provider
# terraform import firezone_oidc_auth_provider.existing <provider-id>

# Query an existing OIDC provider
data "firezone_oidc_auth_provider" "existing" {
  id = "existing-provider-id"
}

output "oidc_provider_name" {
  value = data.firezone_oidc_auth_provider.existing.name
}

# Query an existing Google provider
data "firezone_google_auth_provider" "existing_google" {
  id = firezone_google_auth_provider.google.id
}

output "google_provider_status" {
  value = {
    name       = data.firezone_google_auth_provider.existing_google.name
    is_default = data.firezone_google_auth_provider.existing_google.is_default
    context    = data.firezone_google_auth_provider.existing_google.context
  }
}
