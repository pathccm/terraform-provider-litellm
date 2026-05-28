# litellm_router_settings - Minimal
# Singleton resource — only one per LiteLLM deployment.

resource "litellm_router_settings" "minimal" {
  routing_strategy       = "least-busy"
  num_retries            = 2
  timeout                = 30
  cooldown_time          = 60
  allowed_fails          = 3
  enable_pre_call_checks = true
}

output "router_settings_id" {
  value = litellm_router_settings.minimal.id
}
