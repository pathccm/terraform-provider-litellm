# terraform-provider-litellm — Claude Code Guide

This is a Terraform provider for LiteLLM, written in Go using `terraform-plugin-framework`. It is maintained as a fork at `github.com/pathccm/terraform-provider-litellm` and published to `registry.terraform.io/pathccm/litellm`.

## Keeping the provider current with LiteLLM API changes

### Step 1 — fetch the latest OpenAPI schema

LiteLLM publishes its OpenAPI schema live from any running proxy instance:

```
GET https://<litellm-host>/openapi.json
```

For our deployments:
- Dev: `https://litellm.dev.rula-internal.com/openapi.json`
- Prod: `https://litellm.prod.rula-internal.com/openapi.json`

The schema is also available from the upstream LiteLLM repo at:
`https://raw.githubusercontent.com/BerriAI/litellm/main/schema.prisma` (DB schema) and the proxy OpenAPI at any versioned release tag.

After fetching, replace the checked-in copy:

```bash
curl -s https://litellm.dev.rula-internal.com/openapi.json -o openapi.json
```

Commit the updated `openapi.json` alongside any resource changes so it stays in sync.

### Step 2 — diff the schema against what the provider implements

For each resource, cross-reference the relevant OpenAPI paths against the resource's schema in `internal/provider/resource_<name>.go`. Key things to look for:

| Change type | Where to look in openapi.json | What to update |
|---|---|---|
| New request field | `paths.<endpoint>.requestBody.content.application/json.schema.properties` | Add attribute to schema + `buildRequest()` |
| Removed/renamed field | Same as above | Remove or alias the attribute |
| New response field | `paths.<endpoint>.responses.200.content.application/json.schema.properties` | Add attribute to schema + read mapping |
| Endpoint path changed | `paths` keys | Update the path string in CRUD methods |
| New top-level endpoint | `paths` keys | Consider a new resource or datasource |

For the router settings resource specifically, check `paths./router/settings` for any new fields added to `UpdateRouterConfig`.

### Step 3 — update the affected resource

Each resource follows this structure in `internal/provider/resource_<name>.go`:

1. **`ResourceModel` struct** — one `tfsdk` field per schema attribute
2. **`Schema()`** — declares attributes (Required/Optional/Computed, type, description)
3. **`buildRequest()`** (or inline in Create/Update) — maps model → API request body. Always guard with `IsNull() && IsUnknown()` before setting a field.
4. **`read<Name>()`** — maps API response → model. Use type assertions (`.(string)`, `.(float64)`, etc.) since responses are `map[string]interface{}`.
5. **CRUD methods** — call `DoRequestWithResponse()` with the correct HTTP method and path.

**Adding a new optional field** (most common case):

```go
// 1. Add to the model struct
NewField types.String `tfsdk:"new_field"`

// 2. Add to Schema() attributes
"new_field": schema.StringAttribute{
    Description: "...",
    Optional:    true,
    Computed:    true,
},

// 3. Add to buildRequest()
if !data.NewField.IsNull() && !data.NewField.IsUnknown() {
    req["new_field"] = data.NewField.ValueString()
}

// 4. Add to read<Name>()
if v, ok := result["new_field"].(string); ok {
    data.NewField = types.StringValue(v)
}
```

**Type conventions used in this provider:**

| API type | Go type | tfsdk type |
|---|---|---|
| string | `types.String` | `schema.StringAttribute` |
| integer | `types.Int64` | `schema.Int64Attribute` |
| float/number | `types.Float64` | `schema.Float64Attribute` |
| boolean | `types.Bool` | `schema.BoolAttribute` |
| string list | `types.List` (ElementType: StringType) | `schema.ListAttribute` |
| arbitrary object/map | `types.String` (JSON-encoded) | `schema.StringAttribute` — use `jsonencode()` in HCL |

JSON-encoded map fields (e.g. `routing_strategy_args`, `litellm_params`) follow the pattern in `resource_guardrail.go` and `resource_router_settings.go`. When reading back, use `fmt.Sprintf("%v", v)` for unknown nested types.

JSON numbers from `encoding/json` unmarshal as `float64` — use the `toFloat64()` helper in `resource_router_settings.go` when reading numeric fields.

### Step 4 — update documentation

Each resource has a doc page at `docs/resources/<name>.md`. Update:
- The argument reference table (add/remove/rename attributes)
- The example usage block
- Any notes about version requirements if the new field needs a minimum LiteLLM version

### Step 5 — add a smoke test fixture

Add or update `internal_testing/resources/<name>_minimal.tf` to exercise the new field. Run:

```bash
make local       # start LiteLLM + DB via docker compose
make smoke resources=<name>_minimal.tf
```

### Step 6 — release

Releases are triggered by pushing a `v*` tag. The GitHub Actions workflow in `.github/workflows/release.yml` builds multi-platform binaries via GoReleaser and publishes to the Terraform Registry. Requires `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets set in the GitHub repo.

```bash
git tag v1.x.y
git push origin v1.x.y
```

---

## Repository layout

```
internal/provider/
  client.go                      # DoRequest / DoRequestWithResponse / IsNotFoundError
  provider.go                    # Resources() and DataSources() registration
  resource_<name>.go             # one file per resource
  datasource_<name>.go           # one file per datasource
  resource_router_settings.go    # singleton resource pattern (no delete)

internal_testing/
  docker-compose.yml             # local LiteLLM + Postgres for smoke tests
  resources/                     # smoke test .tf fixtures
  datasources/                   # smoke test datasource fixtures
  provider.tf                    # local dev override provider config

docs/resources/<name>.md         # provider registry documentation
examples/complete/main.tf        # full enterprise example (keep in sync)
openapi.json                     # checked-in LiteLLM OpenAPI schema (update on each LiteLLM version bump)
```

## Development workflow

```bash
make build    # compile the provider binary
make install  # install to ~/.terraform.d/plugins/registry.terraform.io/pathccm/litellm/
make test     # run unit/acceptance tests
make local    # docker compose up for LiteLLM + DB
make smoke resources=<file>.tf   # plan/apply/destroy a fixture
make fmt      # go fmt
make vet      # go vet
```

Local dev override (`~/.terraformrc`):
```hcl
provider_installation {
  dev_overrides {
    "pathccm/litellm" = "/Users/<you>/src/terraform-provider-litellm"
  }
  direct {}
}
```

## Important patterns

- **No terraform-plugin-sdk**: all resources use `terraform-plugin-framework` (v1.17+). The `litellm/` directory contains legacy SDK code — do not add new resources there.
- **Singleton resources** (e.g. `litellm_router_settings`): set `id` to a fixed string, make Delete a no-op, use `ImportState` with `PassthroughID`.
- **Error handling**: use `IsNotFoundError(err)` in Read to remove the resource from state on 404 rather than erroring.
- **Sensitive fields**: mark API keys and secrets `Sensitive: true` in the schema.
- **The `litellm-changed-by` header** is set automatically by the client from the provider's `litellm_changed_by` config — no per-resource handling needed.
