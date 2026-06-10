terraform {
  required_providers {
    fider = {
      source  = "m11s-io/fider"
      version = "~> 0.1"
    }
  }
}

provider "fider" {
  url     = "https://feedback.example.com"
  api_key = var.fider_api_key
}
