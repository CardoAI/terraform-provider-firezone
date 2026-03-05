# Example: Managing Group Memberships
# 
# This example shows how to add users (actors) to groups

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

# Create some actors (users)
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

# Method 1: Add members using PATCH (incremental - add/remove)
# Note: This requires a custom resource or using the API directly
# For now, you need to use the Firezone UI or API to manage memberships

# Method 2: Use the API directly via local-exec (workaround)
resource "null_resource" "add_john_to_engineering" {
  depends_on = [
    firezone_group.engineering,
    firezone_actor.john
  ]

  provisioner "local-exec" {
    command = <<-EOT
      curl -X PATCH \
        -H "Authorization: Bearer ${var.firezone_api_key}" \
        -H "Content-Type: application/json" \
        -d '{"memberships":{"add":["${firezone_actor.john.id}"]}}' \
        "${var.firezone_api_url}/groups/${firezone_group.engineering.id}/memberships"
    EOT
  }

  # Trigger re-run if actor or group changes
  triggers = {
    group_id = firezone_group.engineering.id
    actor_id = firezone_actor.john.id
  }
}

resource "null_resource" "add_jane_to_engineering" {
  depends_on = [
    firezone_group.engineering,
    firezone_actor.jane
  ]

  provisioner "local-exec" {
    command = <<-EOT
      curl -X PATCH \
        -H "Authorization: Bearer ${var.firezone_api_key}" \
        -H "Content-Type: application/json" \
        -d '{"memberships":{"add":["${firezone_actor.jane.id}"]}}' \
        "${var.firezone_api_url}/groups/${firezone_group.engineering.id}/memberships"
    EOT
  }

  triggers = {
    group_id = firezone_group.engineering.id
    actor_id = firezone_actor.jane.id
  }
}

# Method 3: Add multiple members at once
resource "null_resource" "add_multiple_members" {
  depends_on = [
    firezone_group.engineering,
    firezone_actor.john,
    firezone_actor.jane,
    firezone_actor.bob
  ]

  provisioner "local-exec" {
    command = <<-EOT
      curl -X PATCH \
        -H "Authorization: Bearer ${var.firezone_api_key}" \
        -H "Content-Type: application/json" \
        -d '{"memberships":{"add":["${firezone_actor.john.id}","${firezone_actor.jane.id}","${firezone_actor.bob.id}"]}}' \
        "${var.firezone_api_url}/groups/${firezone_group.engineering.id}/memberships"
    EOT
  }

  triggers = {
    group_id   = firezone_group.engineering.id
    member_ids = join(",", [
      firezone_actor.john.id,
      firezone_actor.jane.id,
      firezone_actor.bob.id
    ])
  }
}

# Query group memberships (read-only)
data "external" "engineering_members" {
  depends_on = [null_resource.add_multiple_members]

  program = ["bash", "-c", <<-EOT
    curl -s -H "Authorization: Bearer ${var.firezone_api_key}" \
      "${var.firezone_api_url}/groups/${firezone_group.engineering.id}/memberships" \
      | jq '{members: [.data[].id] | join(",")}'
  EOT
  ]
}

# Create a site and resource for the group to access
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

# Create a policy granting the group access to the resource
resource "firezone_policy" "eng_db_access" {
  description = "Engineering team can access database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.database.id
}

# Outputs
output "group_info" {
  value = {
    group_id   = firezone_group.engineering.id
    group_name = firezone_group.engineering.name
  }
}

output "actors" {
  value = {
    john = {
      id    = firezone_actor.john.id
      email = firezone_actor.john.email
    }
    jane = {
      id    = firezone_actor.jane.id
      email = firezone_actor.jane.email
    }
    bob = {
      id    = firezone_actor.bob.id
      email = firezone_actor.bob.email
    }
  }
}

output "summary" {
  value = <<-EOT
    Group Membership Setup:
    
    Group: ${firezone_group.engineering.name} (${firezone_group.engineering.id})
    
    Members added:
    - ${firezone_actor.john.name} (${firezone_actor.john.email})
    - ${firezone_actor.jane.name} (${firezone_actor.jane.email})
    - ${firezone_actor.bob.name} (${firezone_actor.bob.email})
    
    Access granted:
    - Engineering team → Production Database
    
    Note: Memberships are managed via API calls. 
    A dedicated membership resource will be added in a future version.
  EOT
}
