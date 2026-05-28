# litellm_router_settings (Resource)

Manages global LiteLLM router settings. This is a singleton resource — only one instance should exist per LiteLLM deployment.

**Note:** Destroying this resource removes it from Terraform state only. Router settings have no meaningful API delete; the settings remain on the LiteLLM instance.

## Example Usage

### Minimal

```hcl
resource "litellm_router_settings" "this" {
  routing_strategy       = "least-busy"
  num_retries            = 2
  enable_pre_call_checks = true
}
```

### Full configuration

```hcl
resource "litellm_router_settings" "this" {
  routing_strategy       = "latency-based-routing"
  num_retries            = 3
  max_retries            = 6
  timeout                = 60
  cooldown_time          = 120
  allowed_fails          = 3
  retry_after            = 1.0
  enable_pre_call_checks = true

  model_group_alias = jsonencode({
    "gpt-4" = "gpt-4-enterprise"
  })
}
```

## Argument Reference

### Optional

- `routing_strategy` - (String) The routing strategy. Options: `simple-shuffle`, `least-busy`, `latency-based-routing`, `usage-based-routing`, `cost-based-routing`.
- `routing_strategy_args` - (String) JSON-encoded map of additional arguments for the routing strategy.
- `model_group_retry_policy` - (String) JSON-encoded map of per-model-group retry policies.
- `allowed_fails` - (Number) Number of failures allowed before a model is put in cooldown.
- `cooldown_time` - (Number) Seconds a model stays in cooldown after exceeding `allowed_fails`.
- `num_retries` - (Number) Number of retries per request before marking as failed.
- `timeout` - (Number) Request timeout in seconds.
- `max_retries` - (Number) Maximum total retries across all fallbacks.
- `retry_after` - (Number) Seconds to wait before retrying a failed request.
- `enable_pre_call_checks` - (Boolean) Whether to run pre-call checks (e.g. context window validation) before routing.
- `model_group_alias` - (String) JSON-encoded map of model group aliases.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - Always `router_settings`.

## Import

The router settings singleton can be imported with the fixed ID `router_settings`:

```shell
terraform import litellm_router_settings.this router_settings
```

## Notes

- This resource is a singleton. Define it once per LiteLLM deployment.
- Destroying this resource only removes it from Terraform state; the settings remain active on the LiteLLM instance.
- All attributes are `Optional` and `Computed` — omitting an attribute leaves that setting at its current server value.
