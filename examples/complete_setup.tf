# Complete Firezone setup example with auth providers

terraform {
  required_providers {
    firezone = {
      source = "example/firezone"
    }
  }
}

provider "firezone" {
  api_url = "https://firezone-api.us-services.cardoaiapps.com"
  api_key = var.firezone_api_key
}

# Setup Google authentication as the default provider
resource "firezone_google_auth_provider" "google" {
  name                         = "Google SSO"
  client_id                    = var.google_client_id
  client_secret                = var.google_client_secret
  context                      = "clients_and_portal"
  is_default                   = true
  client_session_lifetime_secs = 86400  # 24 hours
  portal_session_lifetime_secs = 3600   # 1 hour
}

# Setup OIDC provider as a secondary option
resource "firezone_oidc_auth_provider" "keycloak" {
  name                   = "Keycloak"
  client_id              = var.keycloak_client_id
  client_secret          = var.keycloak_client_secret
  discovery_document_uri = "https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration"
  scope                  = "openid email profile"
  context                = "portal_only"
  is_default             = false
}

# Create sites for different regions
resource "firezone_site" "us_east" {
  name = "vpc-us-east"
}

resource "firezone_site" "us_west" {
  name = "vpc-us-west"
}

# Create groups for different teams
resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_group" "devops" {
  name = "DevOps"
}

resource "firezone_group" "data_team" {
  name = "Data Team"
}

# Create actors (users)
resource "firezone_actor" "admin" {
  name                    = "Admin User"
  email                   = "admin@example.com"
  type                    = "account_admin_user"
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "service_account" {
  name                    = "CI/CD Service Account"
  email                   = "cicd@example.com"
  type                    = "service_account"
  allow_email_otp_sign_in = false
}

# Create resources in US East
resource "firezone_resource" "prod_database" {
  name                = "Production Database"
  address             = "10.0.1.10"
  address_description = "Main production PostgreSQL database"
  type                = "ip"
  site_id             = firezone_site.us_east.id
}

resource "firezone_resource" "prod_redis" {
  name                = "Production Redis"
  address             = "10.0.1.20"
  address_description = "Production Redis cache"
  type                = "ip"
  site_id             = firezone_site.us_east.id
}

resource "firezone_resource" "internal_network" {
  name                = "Internal Network"
  address             = "10.0.0.0/16"
  address_description = "Full internal network access"
  type                = "cidr"
  site_id             = firezone_site.us_east.id
}

resource "firezone_resource" "admin_panel" {
  name                = "Admin Panel"
  address             = "admin.internal.example.com"
  address_description = "Internal admin dashboard"
  type                = "dns"
  site_id             = firezone_site.us_east.id
}

# Create resources in US West
resource "firezone_resource" "analytics_db" {
  name                = "Analytics Database"
  address             = "10.1.1.10"
  address_description = "Analytics data warehouse"
  type                = "ip"
  site_id             = firezone_site.us_west.id
}

# Create policies to grant access
resource "firezone_policy" "eng_prod_db" {
  description = "Allow engineering team to access production database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.prod_database.id
}

resource "firezone_policy" "eng_prod_redis" {
  description = "Allow engineering team to access production Redis"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.prod_redis.id
}

resource "firezone_policy" "devops_full_access" {
  description = "Allow DevOps team full internal network access"
  group_id    = firezone_group.devops.id
  resource_id = firezone_resource.internal_network.id
}

resource "firezone_policy" "devops_admin_panel" {
  description = "Allow DevOps team to access admin panel"
  group_id    = firezone_group.devops.id
  resource_id = firezone_resource.admin_panel.id
}

resource "firezone_policy" "data_analytics_db" {
  description = "Allow data team to access analytics database"
  group_id    = firezone_group.data_team.id
  resource_id = firezone_resource.analytics_db.id
}

# Outputs
output "google_auth_provider_id" {
  description = "Google auth provider ID"
  value       = firezone_google_auth_provider.google.id
}

output "sites" {
  description = "Created sites"
  value = {
    us_east = firezone_site.us_east.id
    us_west = firezone_site.us_west.id
  }
}

output "groups" {
  description = "Created groups"
  value = {
    engineering = firezone_group.engineering.id
    devops      = firezone_group.devops.id
    data_team   = firezone_group.data_team.id
  }
}

output "resources_summary" {
  description = "Summary of created resources"
  value = {
    us_east = {
      prod_database     = firezone_resource.prod_database.id
      prod_redis        = firezone_resource.prod_redis.id
      internal_network  = firezone_resource.internal_network.id
      admin_panel       = firezone_resource.admin_panel.id
    }
    us_west = {
      analytics_db = firezone_resource.analytics_db.id
    }
  }
}
