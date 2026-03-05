# Example: Creating Actors (Users and Service Accounts)

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

# Example 1: Create a regular user (account_user)
resource "firezone_actor" "john_doe" {
  name  = "John Doe"
  email = "john.doe@example.com"
  type  = "account_user"
  
  # Optional: Allow email OTP sign-in
  allow_email_otp_sign_in = false
}

# Example 2: Create an admin user (account_admin_user)
resource "firezone_actor" "admin" {
  name  = "Admin User"
  email = "admin@example.com"
  type  = "account_admin_user"
  
  allow_email_otp_sign_in = false
}

# Example 3: Create a service account
resource "firezone_actor" "ci_cd" {
  name  = "CI/CD Pipeline"
  email = "cicd@example.com"
  type  = "service_account"
  
  # Service accounts typically don't use email OTP
  allow_email_otp_sign_in = false
}

# Example 4: Create multiple users with for_each
locals {
  users = {
    "jane" = {
      name  = "Jane Smith"
      email = "jane.smith@example.com"
      type  = "account_user"
    }
    "bob" = {
      name  = "Bob Johnson"
      email = "bob.johnson@example.com"
      type  = "account_user"
    }
    "alice" = {
      name  = "Alice Williams"
      email = "alice.williams@example.com"
      type  = "account_admin_user"
    }
  }
}

resource "firezone_actor" "team_members" {
  for_each = local.users
  
  name                    = each.value.name
  email                   = each.value.email
  type                    = each.value.type
  allow_email_otp_sign_in = false
}

# Example 5: Create service accounts for different purposes
resource "firezone_actor" "monitoring" {
  name  = "Monitoring Service"
  email = "monitoring@example.com"
  type  = "service_account"
  
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "backup" {
  name  = "Backup Service"
  email = "backup@example.com"
  type  = "service_account"
  
  allow_email_otp_sign_in = false
}

resource "firezone_actor" "api_integration" {
  name  = "API Integration"
  email = "api@example.com"
  type  = "service_account"
  
  allow_email_otp_sign_in = false
}

# Example 6: Create actors from a CSV-like data structure
locals {
  csv_users = [
    { name = "Developer 1", email = "dev1@example.com", type = "account_user" },
    { name = "Developer 2", email = "dev2@example.com", type = "account_user" },
    { name = "Developer 3", email = "dev3@example.com", type = "account_user" },
  ]
}

resource "firezone_actor" "developers" {
  count = length(local.csv_users)
  
  name                    = local.csv_users[count.index].name
  email                   = local.csv_users[count.index].email
  type                    = local.csv_users[count.index].type
  allow_email_otp_sign_in = false
}

# Outputs
output "created_actors" {
  description = "Information about created actors"
  value = {
    john_doe = {
      id    = firezone_actor.john_doe.id
      name  = firezone_actor.john_doe.name
      email = firezone_actor.john_doe.email
      type  = firezone_actor.john_doe.type
    }
    admin = {
      id    = firezone_actor.admin.id
      name  = firezone_actor.admin.name
      email = firezone_actor.admin.email
      type  = firezone_actor.admin.type
    }
    ci_cd = {
      id    = firezone_actor.ci_cd.id
      name  = firezone_actor.ci_cd.name
      email = firezone_actor.ci_cd.email
      type  = firezone_actor.ci_cd.type
    }
  }
}

output "team_members" {
  description = "Team member actor IDs"
  value = {
    for key, actor in firezone_actor.team_members : key => {
      id    = actor.id
      name  = actor.name
      email = actor.email
    }
  }
}

output "service_accounts" {
  description = "Service account IDs"
  value = {
    monitoring      = firezone_actor.monitoring.id
    backup          = firezone_actor.backup.id
    api_integration = firezone_actor.api_integration.id
  }
}

output "summary" {
  description = "Summary of created actors"
  value = <<-EOT
    Created Actors:
    
    Regular Users:
    - ${firezone_actor.john_doe.name} (${firezone_actor.john_doe.email})
    - ${length(local.users)} team members
    - ${length(local.csv_users)} developers
    
    Admin Users:
    - ${firezone_actor.admin.name} (${firezone_actor.admin.email})
    
    Service Accounts:
    - ${firezone_actor.ci_cd.name}
    - ${firezone_actor.monitoring.name}
    - ${firezone_actor.backup.name}
    - ${firezone_actor.api_integration.name}
    
    Total: ${1 + 1 + length(local.users) + length(local.csv_users) + 4} actors created
  EOT
}
