---
name: Testing the terraform-provider-netcup SCP resources
description: |
  How to build the netcup Terraform provider from source and exercise the SCP data sources and resources against a local mock SCP API.
---

# Testing the terraform-provider-netcup SCP resources

## Required toolchain

The project uses [mise](https://mise.jdx.dev/). Run `mise install` from the repo root to install Go, Terraform, golangci-lint, lefthook, and `tfplugindocs`.

If mise is not on `PATH`, it was installed for this session at `~/.local/bin/mise`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

The system `go` (`/usr/bin/go`) and `terraform` (`/usr/bin/terraform`) are too old and should not be used. Mise installs the correct versions via `mise.toml`.

## Run the CI checks

```bash
mise run
```

This runs `fmt`, `build`, `test`, and `lint` locally. CI uses `actions/setup-go@v7`,
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
~/.local/share/mise/installs/terraform/1.9.8/terraform -chdir=/tmp/tftest apply -auto-approve -input=false
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
~/.local/share/mise/installs/terraform/1.9.8/terraform -chdir=/tmp/tftest apply -auto-approve -input=false
```

## Common gotchas

- The SCP OpenAPI spec returns camelCase keys; the provider schema is snake_case. If any computed field (e.g. `ipv4addresses`, `server_live_info`, `max_cpu_count`) is `null`, the key-mapping helper in `internal/provider/scpcommon/helpers.go` is not normalizing keys.
- Local checks can be run with `mise run`; CI uses `actions/setup-go@v7` and the
  official GitHub Actions for `golangci-lint` and `GoReleaser` instead of mise.
- Do not run actions against the real netcup SCP API unless a real token is provided and the user explicitly approves destructive operations.
