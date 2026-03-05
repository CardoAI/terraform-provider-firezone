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

# Create a site
resource "firezone_site" "main" {
  name = "vpc-us-east"
}

# Create a group
resource "firezone_group" "engineering" {
  name = "Engineering"
}

# Create an actor
resource "firezone_actor" "john" {
  name                    = "John Doe"
  email                   = "john.doe@example.com"
  type                    = "account_user"
  allow_email_otp_sign_in = false
}

# Create a resource
resource "firezone_resource" "database" {
  name                = "Production Database"
  address             = "10.0.0.10"
  address_description = "Main production database"
  type                = "ip"
  site_id             = firezone_site.main.id
}

# Create a policy
resource "firezone_policy" "eng_db_access" {
  description = "Allow engineering team to access production database"
  group_id    = firezone_group.engineering.id
  resource_id = firezone_resource.database.id
}

# Data source example
data "firezone_group" "existing" {
  id = "existing-group-id"
}

output "group_name" {
  value = data.firezone_group.existing.name
}
