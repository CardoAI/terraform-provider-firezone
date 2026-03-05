# Example: Managing Group Memberships (Clean Method)
# Using the firezone_group_membership resource

terraform {
  required_providers {
    firezone = {
      source  = "registry.terraform.io/example/firezone"
      version = "0.1.0"
    }
  }
}

provider "firezone" {
  api_url = var.firezone_api_url
  api_key = var.firezone_api_key
}

# Create a group
resource "firezone_group" "engineering" {
  name = "Engineering"
}

resource "firezone_group" "devops" {
  name = "DevOps"
}

# Create actors (users)
resource "firezone_actor" "john" {
  name                    = "John Doe"
  email                   = "john.doe@example.com"
  type                    = "account_user"
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "jane" {
  name                    = "Jane Smith"
  email                   = "jane.smith@example.com"
  type                    = "account_user"
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "bob" {
  name                    = "Bob Johnson"
  email                   = "bob.johnson@example.com"
  type                    = "account_admin_user"
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "alice" {
  name                    = "Alice Williams"
  email                   = "alice.williams@example.com"
  type                    = "account_user"
  allow_email_otp_sign_in = false
}

# Add members to Engineering group
resource "firezone_group_membership" "engineering_members" {
  group_id = firezone_group.engineering.id
  
  actor_ids = [
    firezone_actor.john.id,
    firezone_actor.jane.id,
    firezone_actor.alice.id,
  ]
}

# Add members to DevOps group
resource "firezone_group_membership" "devops_members" {
  group_id = firezone_group.devops.id
  
  actor_ids = [
    firezone_actor.bob.id,
    firezone_actor.alice.id,  # Alice is in both groups
  ]
}

# Create resources
resource "firezone_site" "main" {
  name = "main-site"
}

resource "firezone_resource" "database" {
  name                = "Production Database"
  address             = "10.0.1.10"
  address_description = "Main database"
  type                = "ip"
  site_id             = firezone_site.main.id
}

resource "firezone_resource" "api" {
  name                = "Internal API"
  address             = "api.internal.example.com"
  address_description = "Internal REST API"
  type                = "dns"
  site_id             = firezone_site.main.id
}

# Create policies
resource "firezone_policy" "eng_db_access" {
  description = "Engineering team can access database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.database.id
}

resource "firezone_policy" "devops_full_access" {
  description = "DevOps team can access everything"
  group_id    = firezone_group.devops.id
  resource_id = firezone_resource.api.id
}

# Outputs
output "engineering_group" {
  value = {
    id      = firezone_group.engineering.id
    name    = firezone_group.engineering.name
    members = firezone_group_membership.engineering_members.actor_ids
  }
}

output "devops_group" {
  value = {
    id      = firezone_group.devops.id
    name    = firezone_group.devops.name
    members = firezone_group_membership.devops_members.actor_ids
  }
}

output "summary" {
  value = <<-EOT
    Group Memberships:
    
    Engineering (${firezone_group.engineering.id}):
    - ${firezone_actor.john.name} (${firezone_actor.john.email})
    - ${firezone_actor.jane.name} (${firezone_actor.jane.email})
    - ${firezone_actor.alice.name} (${firezone_actor.alice.email})
    
    DevOps (${firezone_group.devops.id}):
    - ${firezone_actor.bob.name} (${firezone_actor.bob.email})
    - ${firezone_actor.alice.name} (${firezone_actor.alice.email})
    
    Access Policies:
    - Engineering → Production Database
    - DevOps → Internal API
  EOT
}
