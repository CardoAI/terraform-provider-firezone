variable "google_client_id" {
  description = "Google OAuth2 Client ID"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth2 Client Secret"
  type        = string
  sensitive   = true
}

variable "keycloak_client_id" {
  description = "Keycloak OAuth2 Client ID"
  type        = string
  sensitive   = true
}

variable "keycloak_client_secret" {
  description = "Keycloak OAuth2 Client Secret"
  type        = string
  sensitive   = true
}
