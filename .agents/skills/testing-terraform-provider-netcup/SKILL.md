---
name: Testing the terraform-provider-netcup SCP resources
description: |
  How to build the netcup Terraform provider from source and exercise the SCP data sources and resources against a local mock SCP API.
---

# Testing the terraform-provider-netcup SCP resources

## Required toolchain

- Go 1.26.5 at `/usr/local/go/bin/go` (the system `go` is too old).
- Terraform CLI. The HashiCorp apt package on this box reports `v1.15.8` but behaves unreliably with dev overrides; download a known-good Linux amd64 binary (e.g. `1.9.8`) and run it from `/tmp/tf-bin/terraform`.
- Python 3 for the mock SCP API.

## Build the provider binary

```bash
export PATH=/usr/local/go/bin:$PATH
CGO_ENABLED=0 go build -o /tmp/terraform-provider-netcup .
```

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
/tmp/tf-bin/terraform -chdir=/tmp/tftest apply -auto-approve -input=false
```

## Common gotchas

- The SCP OpenAPI spec returns camelCase keys; the provider schema is snake_case. If any computed field (e.g. `ipv4addresses`, `server_live_info`, `max_cpu_count`) is `null`, the key-mapping helper in `internal/provider/scpcommon/helpers.go` is not normalizing keys.
- `go build`, `go vet`, and `go test ./...` should all be run with `/usr/local/go/bin/go` and pass.
- Do not run actions against the real netcup SCP API unless a real token is provided and the user explicitly approves destructive operations.
