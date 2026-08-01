---
name: Testing the terraform-provider-netcup SCP resources
description: |
  How to build the netcup Terraform provider from source and exercise the SCP data sources and resources against a local mock SCP API.
---

# Testing the terraform-provider-netcup SCP resources

## Required toolchain

The project uses [mise](https://mise.jdx.dev/). Run `mise install` from the repo root to install Go, Terraform, golangci-lint, lefthook, `goreleaser`, and `tfplugindocs`. `mise.toml` currently pins `terraform = "1.15.8"` and `goreleaser = "2.17.1"`.

If mise is not on `PATH`, it was installed for this session at `~/.local/bin/mise`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

The system `go` (`/usr/bin/go`) and `terraform` (`/usr/bin/terraform`) are too old and should not be used. Mise installs the correct versions via `mise.toml`.

## Run the CI checks

```bash
mise run
```

This runs `fmt`, `build`, `test`, and `lint` locally. For acceptance tests run `mise run testacc` (or `TF_ACC=1 go test -timeout=120m ./internal/provider/...`). CI uses `actions/setup-go@v7`,
`golangci/golangci-lint-action@v9`, and `goreleaser/goreleaser-action` instead of mise.

## Build the provider binary

```bash
mise run build
```

The binary is produced at the module root (`terraform-provider-netcup`). To place it in `/tmp`:

```bash
CGO_ENABLED=0 mise exec -- go build -o /tmp/terraform-provider-netcup .
```

## Generate Terraform Registry docs

```bash
mise run docs
```

`git diff --stat` should be empty after generation.

## Snapshot release build

```bash
mise exec -- goreleaser build --snapshot --clean
```

Artifacts land in `dist/`. Confirm `dist/terraform-provider-netcup_windows_arm64_v8.0/` is produced.

## Configure Terraform to use the local binary

Create `/tmp/tftest/terraform.rc`:

```hcl
provider_installation {
  dev_overrides {
    "rixlhq/netcup" = "/tmp"
  }
  direct {}
}
```

Important: when dev overrides are active, `terraform init` may fail to resolve the provider version from the registry. You can skip `terraform init` entirely and run `terraform plan`/`apply` directly against the local binary.

Use the mise-managed Terraform binary. Because the system `/usr/bin/terraform` may shadow the mise shim in this environment, use the direct install path:

```bash
~/.local/share/mise/installs/terraform/1.15.8/terraform -chdir=/tmp/tftest apply -auto-approve -input=false
```

## Start a local mock SCP API

Use the Python `http.server` example in `/tmp/mock_scp.py`. It must:

- respond to `GET /api/v1/servers` with a `ServerListMinimal` JSON array,
- respond to `GET /api/v1/servers/{id}` with a full `Server` JSON object using camelCase keys (the provider converts them to snake_case),
- accept `PATCH /api/v1/servers/{id}` and return the updated server,
- accept the action routes you intend to test (e.g. `POST /api/v1/servers/{id}/storageoptimization`).

The mock writes captured requests to `/tmp/mock_scp_requests.log`.

## Example Terraform root module

```hcl
terraform {
  required_providers {
    netcup = { source = "rixlhq/netcup" }
  }
}

provider "netcup" {
  scp_access_token = "test-token"
  scp_base_url     = "http://127.0.0.1:<MOCK_PORT>"
}

data "netcup_scp_server" "example" {
  server_id = 12345
}

data "netcup_scp_servers" "all" {}
```

Run with:

```bash
export TF_CLI_CONFIG_FILE=/tmp/tftest/terraform.rc
~/.local/share/mise/installs/terraform/1.15.8/terraform -chdir=/tmp/tftest apply -auto-approve -input=false
```

## Environment-variable credential fallback

To verify provider credentials can be supplied via environment variables, build the provider binary and run Terraform with an empty `provider` block:

```bash
CGO_ENABLED=0 mise exec -- go build -o /tmp/terraform-provider-netcup .

# In one terminal, start a mock SCP API on /scp-core/api/v1/rdns/ipv4, etc.
python3 /tmp/mock_scp_env.py

# In another terminal:
export TF_CLI_CONFIG_FILE=/tmp/tftest_env/terraform.rc
export NETCUP_SCP_ACCESS_TOKEN='env-token'
export NETCUP_SCP_BASE_URL='http://127.0.0.1:<MOCK_PORT>/scp-core'

~/.local/share/mise/installs/terraform/1.15.8/terraform -chdir=/tmp/tftest_env apply -auto-approve
~/.local/share/mise/installs/terraform/1.15.8/terraform -chdir=/tmp/tftest_env plan
~/.local/share/mise/installs/terraform/1.15.8/terraform -chdir=/tmp/tftest_env destroy -auto-approve
```

Pass criteria:

- `terraform apply` creates `netcup_scp_rdns`.
- `terraform plan` reports `No changes.`.
- `terraform destroy` succeeds.
- The mock server receives `Authorization: Bearer env-token`, proving env vars were used.

## Acceptance test coverage

- `TestAccScpRdns_basic` validates the `netcup_scp_rdns` resource against an `httptest` mock.
- `TestAccScpServerInterfaceFirewall_basic` validates the `netcup_scp_server_interface_firewall` resource and the `scpclient` 202/`FINISHED` task short-circuit behavior.

## Common gotchas

- The SCP OpenAPI spec returns camelCase keys; the provider schema is snake_case. If any computed field (e.g. `ipv4addresses`, `server_live_info`, `max_cpu_count`) is `null`, the key-mapping helper in `internal/provider/scpcommon/helpers.go` is not normalizing keys.
- Local checks can be run with `mise run`; CI uses `actions/setup-go@v7` and the
  official GitHub Actions for `golangci-lint` and `GoReleaser` instead of mise.
- Do not run actions against the real netcup SCP API unless a real token is provided and the user explicitly approves destructive operations.
- `mise run testacc` runs acceptance tests with the local mock SCP server; no real credentials are needed for `TestAccScpRdns_basic` or `TestAccScpServerInterfaceFirewall_basic`.
