# Example: Querying existing Firezone resources

# Query an existing OIDC auth provider
data "firezone_oidc_auth_provider" "existing_oidc" {
  id = "your-oidc-provider-id"
}

# Use the queried data
output "oidc_provider_info" {
  value = {
    name                   = data.firezone_oidc_auth_provider.existing_oidc.name
    client_id              = data.firezone_oidc_auth_provider.existing_oidc.client_id
    discovery_document_uri = data.firezone_oidc_auth_provider.existing_oidc.discovery_document_uri
    scope                  = data.firezone_oidc_auth_provider.existing_oidc.scope
    is_default             = data.firezone_oidc_auth_provider.existing_oidc.is_default
    is_disabled            = data.firezone_oidc_auth_provider.existing_oidc.is_disabled
  }
}

# Query an existing Google auth provider
data "firezone_google_auth_provider" "existing_google" {
  id = "your-google-provider-id"
}

output "google_provider_info" {
  value = {
    name        = data.firezone_google_auth_provider.existing_google.name
    client_id   = data.firezone_google_auth_provider.existing_google.client_id
    is_default  = data.firezone_google_auth_provider.existing_google.is_default
    is_disabled = data.firezone_google_auth_provider.existing_google.is_disabled
  }
}

# Query existing groups
data "firezone_group" "engineering" {
  id = "your-group-id"
}

# Query existing actors
data "firezone_actor" "admin" {
  id = "your-actor-id"
}

# Use queried data to create new resources
resource "firezone_policy" "new_policy" {
  description = "New policy using existing group"
  group_id    = data.firezone_group.engineering.id
  resource_id = firezone_resource.new_resource.id
}

resource "firezone_resource" "new_resource" {
  name                = "New Resource"
  address             = "10.0.2.50"
  address_description = "Newly created resource"
  type                = "ip"
  site_id             = "your-site-id"
}

# Conditional logic based on provider status
locals {
  google_is_active = data.firezone_google_auth_provider.existing_google.is_default && !data.firezone_google_auth_provider.existing_google.is_disabled
}

output "google_auth_status" {
  value = local.google_is_active ? "Active and default" : "Not active or not default"
}
