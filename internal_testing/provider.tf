terraform {
  required_providers {
    litellm = {
      source = "pathccm/litellm"
    }
  }
}

provider "litellm" {
  api_base = var.litellm_api_base
  api_key  = var.litellm_api_key
}
