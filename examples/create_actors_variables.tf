variable "firezone_api_url" {
  description = "Firezone API URL"
  type        = string
  default     = "https://firezone-api.us-services.cardoaiapps.com"
}

variable "firezone_api_key" {
  description = "Firezone API Key"
  type        = string
  sensitive   = true
}
