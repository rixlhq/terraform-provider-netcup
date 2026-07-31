# Terraform Provider for Netcup

A Terraform provider for managing resources at [netcup](https://www.netcup.com).
It supports both the **Customer Control Panel (CCP) DNS API** and the **Server Control Panel (SCP) REST API**.

- DNS records and zones are managed through the CCP JSON API.
- Servers, networks, snapshots, ISOs, tasks, users, and other SCP resources are
  exposed as data sources using the SCP OpenAPI specification.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.26

## Provider Configuration

Configure the credentials for the APIs you want to use. CCP credentials are
required for DNS resources, and an SCP access token is required for SCP data
sources. Both can be configured at the same time.

```hcl
terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 0.1"
    }
  }
}

provider "netcup" {
  api_key         = "my-api-key"
  api_password    = "my-api-password"
  customer_number = "123456"

  scp_access_token  = "my-bearer-token"
  scp_refresh_token = "my-refresh-token" # optional, for automatic refresh
}
```

CCP credentials are generated in the netcup Customer Control Panel under **Master Data > API**.
SCP tokens are generated via the netcup SCP device-code OAuth flow.

## CCP DNS Resources

### `netcup_dns_record`

Manage a single DNS record. The provider appends, updates, and removes records
using the `updateDnsRecords` API endpoint.

```hcl
resource "netcup_dns_record" "www" {
  zone        = "example.com"
  hostname    = "www"
  type        = "CNAME"
  destination = "example.com"
}

resource "netcup_dns_record" "mx" {
  zone        = "example.com"
  hostname    = "@"
  type        = "MX"
  destination = "mail.example.com"
  priority    = 10
}
```

### `netcup_dns_zone`

Manage the TTL of an existing DNS zone.

```hcl
resource "netcup_dns_zone" "example" {
  domain_name = "example.com"
  ttl         = 3600
}
```

## CCP DNS Data Sources

### `netcup_dns_records`

Read all records of a DNS zone.

```hcl
data "netcup_dns_records" "example" {
  domain_name = "example.com"
}
```

### `netcup_dns_zone`

Read DNS zone metadata.

```hcl
data "netcup_dns_zone" "example" {
  domain_name = "example.com"
}
```

## SCP Data Sources

The SCP data sources map the netcup SCP OpenAPI specification. Examples:

```hcl
data "netcup_scp_server" "example" {
  server_id = 12345
}

data "netcup_scp_servers" "all" {
  limit = 10
}

data "netcup_scp_user" "me" {
  user_id = "me"
}
```

All SCP `GET` endpoints are available as data sources with names derived from
their path, e.g. `netcup_scp_server_interfaces`, `netcup_scp_server_snapshots`,
`netcup_scp_rdns_ipv4`, `netcup_scp_tasks`, `netcup_scp_user_images`, etc.

## SCP Resources

### `netcup_scp_server`

Manage mutable attributes of an existing SCP server. The server cannot be created
or deleted via the SCP API; this resource adopts an existing server by id and
applies patches.

```hcl
resource "netcup_scp_server" "example" {
  server_id = 12345

  hostname = "srv.example.com"
  nickname = "My Server"
  autostart = true
  uefi      = false

  bootorder = ["HDD", "CDROM", "NETWORK"]
  os_optimization = "LINUX"
  keyboard_layout = "de"

  cpu_topology = {
    socket_count           = 1
    cores_per_socket_count = 2
  }
}
```

### `netcup_scp_server_action`

Trigger one-off server actions such as `start`, `stop`, `reset`, `powercycle`,
`suspend`, `rescue_activate`, `snapshot_create`, `snapshot_revert`,
`iso_attach`, `image_setup`, `disk_format`, `firewall_reapply`, and others.

```hcl
resource "netcup_scp_server_action" "start" {
  server_id = 12345
  action    = "start"
}

resource "netcup_scp_server_action" "snapshot" {
  server_id = 12345
  action    = "snapshot_create"

  body = jsonencode({
    name   = "before-upgrade"
    online = true
  })
}
```

Use the `triggers` map to force an action to run again when needed.

## Limitations

- Netcup does not support per-record TTLs; TTL is set per zone. Use
  `netcup_dns_zone` to change it.
- Creating or deleting DNS zones is not supported by the CCP API; only records
  within an existing zone can be managed.
- SCP servers cannot be created or deleted through the SCP API; the
  `netcup_scp_server` resource manages attributes of an existing server.

## Development

Build the provider:

```sh
go build ./...
```

Run unit tests:

```sh
go test ./...
```

Run CCP acceptance tests (requires a real netcup account and environment variables):

```sh
NETCUP_CUSTOMER_NUMBER=... \
NETCUP_API_KEY=... \
NETCUP_API_PASSWORD=... \
TF_ACC=1 go test ./...
```

Run SCP tests (read-only data sources):

```sh
NETCUP_SCP_ACCESS_TOKEN=... \
TF_ACC=1 go test ./internal/provider/scp/...
```

## License

[MIT](./LICENSE)
