# Terraform Provider for Netcup

[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-rixlhq%2Fnetcup-844FBA?logo=terraform&logoColor=white)](https://registry.terraform.io/providers/rixlhq/netcup/latest)
[![Release](https://img.shields.io/github/v/release/rixlhq/terraform-provider-netcup?sort=semver)](https://github.com/rixlhq/terraform-provider-netcup/releases)
[![CI](https://github.com/rixlhq/terraform-provider-netcup/actions/workflows/ci.yml/badge.svg)](https://github.com/rixlhq/terraform-provider-netcup/actions/workflows/ci.yml)
[![License: MPL-2.0](https://img.shields.io/github/license/rixlhq/terraform-provider-netcup)](LICENSE)

A [Terraform](https://www.terraform.io) provider for managing infrastructure at [netcup](https://www.netcup.com) via official netcup APIs:

- **CloudDNS REST API** (Bearer API) — manage individual CloudDNS records and read domain metadata.
- **Customer Control Panel (CCP) DNS API / DomainWebservice** (JSON API) — manage DNS zones and individual DNS records.
- **Server Control Panel (SCP) REST API** (OpenAPI / OAuth 2.0) — manage virtual servers (VPS / Root Server), reverse DNS (rDNS IPv4 & IPv6), network interfaces, VLANs, failover IPs, firewall policies, snapshots, custom disk images, and ISOs.

Built with the modern **Terraform Plugin Framework** (protocol version 6), supporting ephemeral token exchange (`netcup_scp_access_token`) and provider functions (`provider::netcup::ip_version`).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 (>= 1.10 for ephemeral resources, >= 1.8 for provider-defined functions)
- [Go](https://go.dev/doc/install) >= 1.26
- [mise](https://mise.jdx.dev/) (recommended for running tasks and installing tools)

## Overview of Resources & Data Sources

| Category | API | Resources | Data Sources |
| :--- | :--- | :--- | :--- |
| **CloudDNS** | CloudDNS REST API | `netcup_clouddns_record` | `netcup_clouddns_domains`, `netcup_clouddns_records` |
| **DNS (Legacy CCP)** | CCP DNS API | `netcup_dns_record`, `netcup_dns_zone` | `netcup_dns_records`, `netcup_dns_zone` |
| **Servers** | SCP REST API | `netcup_scp_server`, `netcup_scp_server_action` | `netcup_scp_server`, `netcup_scp_servers`, `netcup_scp_server_metrics`, `netcup_scp_server_logs` |
| **Networking & rDNS** | SCP REST API | `netcup_scp_rdns`, `netcup_scp_server_interface`, `netcup_scp_failover_ip_v4`, `netcup_scp_failover_ip_v6`, `netcup_scp_user_vlan` | `netcup_scp_rdns_ipv4`, `netcup_scp_rdns_ipv6`, `netcup_scp_server_interface`, `netcup_scp_server_interfaces`, `netcup_scp_user_failoverips_v4`, `netcup_scp_user_failoverips_v6`, `netcup_scp_user_vlan`, `netcup_scp_user_vlans`, `netcup_scp_vlan` |
| **Firewall** | SCP REST API | `netcup_scp_user_firewall_policy`, `netcup_scp_server_interface_firewall` | `netcup_scp_user_firewall_policy`, `netcup_scp_user_firewall_policies` |
| **Storage & Snapshots** | SCP REST API | `netcup_scp_server_snapshot` | `netcup_scp_server_snapshot`, `netcup_scp_server_snapshots`, `netcup_scp_server_disk`, `netcup_scp_server_disks`, `netcup_scp_server_disk_supported_drivers`, `netcup_scp_server_imageflavours`, `netcup_scp_server_iso`, `netcup_scp_server_isoimages`, `netcup_scp_user_image`, `netcup_scp_user_image_part`, `netcup_scp_user_images`, `netcup_scp_user_iso`, `netcup_scp_user_iso_part`, `netcup_scp_user_isos` |
| **User & Access** | SCP REST API | `netcup_scp_user`, `netcup_scp_user_ssh_key`, `netcup_scp_access_token` (ephemeral) | `netcup_scp_user`, `netcup_scp_user_ssh_keys`, `netcup_scp_user_logs` |
| **Tasks & Status** | SCP REST API | `netcup_scp_task_action` | `netcup_scp_task`, `netcup_scp_tasks`, `netcup_scp_maintenance`, `netcup_scp_server_guest_agent`, `netcup_scp_server_guest_agent_status`, `netcup_scp_server_gpu_driver`, `netcup_scp_server_rescuesystem` |

## Provider Configuration

Configure the credentials for the APIs you want to use. CCP credentials are
required for `netcup_dns_*`, a CloudDNS API key for `netcup_clouddns_*`, and either an SCP access token or refresh token is
required for SCP resources and data sources. Both SCP token types can be configured at the
same time.

```hcl
terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 1.0"
    }
  }
}

provider "netcup" {
  clouddns_api_key = var.netcup_clouddns_api_key # optional: CloudDNS domains

  api_key         = "my-api-key"
  api_password    = "my-api-password"
  customer_number = "123456"

  scp_access_token  = "my-bearer-token"
  scp_refresh_token = "my-refresh-token" # optional, for automatic refresh
}
```

CloudDNS keys are generated in **CCP > Master Data > API > API Keys**.
Legacy CCP credentials come from **Legacy API Keys** in the same section.
SCP tokens are generated via the netcup SCP device-code OAuth flow.

## Protocol Version 6 Features

The provider is built on Terraform Plugin Framework protocol version 6 and exposes
features that require newer Terraform CLI versions:

- **Provider-defined functions** (`Terraform >= 1.8`):
  `provider::netcup::ip_version(ip)` returns `ipv4` or `ipv6` for a given address.
- **Ephemeral resources** (`Terraform >= 1.10`):
  `netcup_scp_access_token` exchanges a refresh token for a short-lived access token
  that is never stored in Terraform state or plan files.

### `netcup_scp_access_token` ephemeral resource

```hcl
ephemeral "netcup_scp_access_token" "token" {
  refresh_token = var.netcup_scp_refresh_token
}

provider "netcup" {
  scp_access_token = ephemeral.netcup_scp_access_token.token.access_token
}
```

### `provider::netcup::ip_version` function

```hcl
resource "netcup_scp_rdns" "example" {
  ip_version = provider::netcup::ip_version("1.2.3.4")
  ip         = "1.2.3.4"
  rdns       = "host.example.com"
}
```

## DNS API compatibility

The `netcup_dns_*` resources and data sources use netcup's **legacy CCP DNS
API** with `api_key`, `api_password`, and `customer_number`. SCP access and
refresh tokens authorize server operations and cannot be used for DNS requests.

Domains using **CloudDNS** require the separate [netcup API](https://www.netcup.com/en/helpcenter/documentation/general/netcup-api),
using `clouddns_api_key` or `NETCUP_CLOUDDNS_API_KEY`. Use the dedicated
`netcup_clouddns_*` resource and data sources. Check the domain's DNS or
CloudDNS tab in the CCP before choosing the API.

## CloudDNS resources and data sources

`netcup_clouddns_record` manages a single record with strict, automatically
committed changesets. An update deletes the previous record and creates its
replacement in one changeset; unrelated records are preserved. Existing records
must be imported, and system-managed immutable records cannot be changed.
Do not manage the same record from multiple Terraform resources.

```hcl
# Supply NETCUP_CLOUDDNS_API_KEY through your secret manager.
provider "netcup" {}

data "netcup_clouddns_domains" "all" {}

resource "netcup_clouddns_record" "www" {
  domain_id = one([
    for domain in data.netcup_clouddns_domains.all.domains : domain.id
    if domain.fqdn == "example.com" && domain.is_dns_managed
  ])
  name  = "www"
  type  = "A"
  rdata = "192.0.2.10"
  ttl   = 300
}

data "netcup_clouddns_records" "example" {
  domain_id = netcup_clouddns_record.www.domain_id
  depends_on = [netcup_clouddns_record.www]
}
```

`rdata` follows the API's DNS presentation format: MX includes the priority
(e.g. `10 mail.example.com.`), and SRV includes priority, weight, port, and target.
TTL is required and must be at least one second. Data sources preserve null TTLs
on system records. Reads select the active revision and never create drafts.

Import with `terraform import netcup_clouddns_record.www 'domain_id/record_identifier'`.
Record identifiers are available in `netcup_clouddns_records`. CloudDNS support
does not register domains or migrate legacy DNS zones.

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

data "netcup_scp_user" "example" {
  user_id = 12345
}
```

All SCP `GET` endpoints are available as data sources with names derived from
their path, e.g. `netcup_scp_server_interfaces`, `netcup_scp_server_snapshots`,
`netcup_scp_rdns_ipv4`, `netcup_scp_tasks`, `netcup_scp_user_images`, etc.

### `netcup_scp_server_imageflavours` and `netcup_scp_server_isoimages`

List the public image flavours and ISO images available to a server. These
are used when calling `image_setup` or `iso_attach` actions.

```hcl
data "netcup_scp_server_imageflavours" "example" {
  server_id = 12345
}

data "netcup_scp_server_isoimages" "example" {
  server_id = 12345
}
```

### `netcup_scp_user_images` and `netcup_scp_user_isos`

List custom images and ISOs uploaded to the account. Use `key` in
`user_image_setup` or `iso_attach` actions.

```hcl
data "netcup_scp_user" "example" {
  user_id = 12345
}

data "netcup_scp_user_images" "example" {
  user_id = data.netcup_scp_user.example.id
}

data "netcup_scp_user_isos" "example" {
  user_id = data.netcup_scp_user.example.id
}
```

### `netcup_scp_server_metrics`

Reads raw server metrics for `cpu`, `disk`, `network` or `network_packet`.

```hcl
data "netcup_scp_server_metrics" "cpu" {
  server_id = 12345
  metric    = "cpu"
  hours     = 1
}

locals {
  cpu_metrics = jsondecode(data.netcup_scp_server_metrics.cpu.json)
}
```

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
`iso_attach`, `image_setup`, `user_image_setup`, `disk_format`,
`firewall_reapply`, and others.

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
    onlineSnapshot = true
  })
}

# Install a public image flavour.
resource "netcup_scp_server_action" "image_setup" {
  server_id = 12345
  action    = "image_setup"

  body = jsonencode({
    imageFlavourId = 123
    diskName       = "sda"
    hostname       = "srv.example.com"
    locale         = "en_US.UTF-8"
    timezone       = "Europe/Berlin"
  })
}

# Install a custom user image by key.
resource "netcup_scp_server_action" "user_image_setup" {
  server_id = 12345
  action    = "user_image_setup"

  body = jsonencode({
    userImageName = "my-custom-image"
    diskName      = "sda"
  })
}

# Attach an ISO image.
resource "netcup_scp_server_action" "iso_attach" {
  server_id = 12345
  action    = "iso_attach"

  body = jsonencode({
    isoId                   = 456
    changeBootDeviceToCdrom = true
  })
}
```

Use the `triggers` map to force an action to run again when needed.

### `netcup_scp_server_snapshot`

Manages server snapshots.

```hcl
resource "netcup_scp_server_snapshot" "example" {
  server_id = 12345
  name      = "before-upgrade"
  online_snapshot = true
}
```

### `netcup_scp_rdns`

Manages reverse DNS entries for IPv4 and IPv6.

```hcl
resource "netcup_scp_rdns" "example" {
  ip_version = "ipv4"
  ip         = "192.0.2.1"
  rdns       = "srv.example.com"
}
```

### `netcup_scp_user_firewall_policy`

Manages account-level firewall policies.

```hcl
resource "netcup_scp_user_firewall_policy" "example" {
  user_id = 12345
  name    = "allow-ssh"

  rules = [
    {
      action      = "ACCEPT"
      direction   = "INGRESS"
      protocol    = "TCP"
      source_ports = "22"
    }
  ]
}
```

### `netcup_scp_failover_ip_v4` / `netcup_scp_failover_ip_v6`

Routes an existing failover IP to a server. Failover IPs cannot be created or
deleted; only the `server_id` target can be changed.

```hcl
resource "netcup_scp_failover_ip_v4" "example" {
  user_id        = 12345
  failover_ip_id = 67890
  server_id      = 54321
}
```

### `netcup_scp_user_vlan`

Updates the name of an existing user VLAN.

```hcl
resource "netcup_scp_user_vlan" "example" {
  user_id = 12345
  vlan_id = 67890
  name    = "my-vlan"
}
```

### `netcup_scp_user`

Updates an existing SCP user account. Users cannot be created or deleted
through the API; this resource adopts the user by `user_id`.

```hcl
resource "netcup_scp_user" "example" {
  user_id   = 12345
  language  = "en"
  time_zone = "Europe/Berlin"

  show_nickname     = true
  passwordless_mode = false
  secure_mode       = true
}
```

### `netcup_scp_user_ssh_key`

Manages SSH keys for an SCP user account.

```hcl
resource "netcup_scp_user_ssh_key" "example" {
  user_id = 12345
  name    = "my-key"
  key     = "ssh-ed25519 AAAAC3NzaC... example"
}
```

### `netcup_scp_task_action`

Triggers one-off task actions. Currently only `cancel` is supported.

```hcl
resource "netcup_scp_task_action" "cancel" {
  task_uuid = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  action    = "cancel"
}
```

### `netcup_scp_server_interface`

Manages a network interface for an SCP server. The interface is created
with a VLAN and network driver; the MAC address is returned by the SCP
task result and used as the Terraform ID. Import via `server_id/mac`.

```hcl
resource "netcup_scp_server_interface" "example" {
  server_id      = 12345
  vlan_id        = 100
  network_driver = "virtio"
}
```

### `netcup_scp_server_interface_firewall`

Manages the firewall configuration for a server network interface.
Deleting the resource deactivates the firewall (`active = false`) since
the SCP API has no delete endpoint. Import via `server_id/mac`.

```hcl
resource "netcup_scp_server_interface_firewall" "example" {
  server_id = 12345
  mac       = "00:50:56:00:00:01"
  active    = true

  copied_policy_ids = [1, 2]
  user_policy_ids   = [10]
}
```

## Limitations

- Netcup does not support per-record TTLs; TTL is set per zone. Use
  `netcup_dns_zone` to change it.
- Creating or deleting DNS zones is not supported by the CCP API; only records
  within an existing zone can be managed.
- SCP servers, failover IPs and VLANs cannot be created or deleted through the
  SCP API. Use the corresponding resources to adopt and update existing objects.
- `netcup_scp_server_interface_firewall` has no delete endpoint in the SCP
  API; destroying the resource deactivates the firewall instead.

## Publishing to the Terraform Registry

The repository is set up to publish releases that the Terraform Registry can
consume.

- The provider address is `registry.terraform.io/rixlhq/netcup`.
- `main.go` injects the version at build time.
- `terraform-registry-manifest.json` declares protocol version `6.0`.
- `.goreleaser.yml` builds cross-platform archives, a SHA256 checksum file, a
detached GPG signature, and renames the manifest to the required asset name.
- `.github/workflows/ci.yml` runs lint, test, and build in parallel steps on every
  push and pull request.
- `.github/workflows/release.yml` runs release-please on successful CI, then
  runs `goreleaser/goreleaser-action` to create a signed release when a
  release is created.

To publish:
1. Generate a GPG signing key and add the public key to the Terraform Registry.
2. Add the private key and passphrase as repository secrets named
   `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`.
3. Create the provider in the Terraform Registry, point it at this GitHub
   repository, and configure the webhook if needed.
4. Merge a conventional commit to `main`; release-please opens a release PR
   and tags the release (e.g. `v1.2.1`) on merge.

## Development

This project uses [mise](https://mise.jdx.dev/) to manage tools and tasks. The
`mise.toml` file pins Go, Terraform, golangci-lint, lefthook, and `tfplugindocs`.
[lefthook](https://github.com/evilmartians/lefthook) is configured for pre-commit
and pre-push checks.

Install tools and set up git hooks:

```sh
mise install
```

Run the default local checks (format, build, test, lint):

```sh
mise run
```

Pre-commit and pre-push hooks run automatically after `mise install`. To run them
manually:

```sh
lefthook run pre-commit
lefthook run pre-push
```

Build the provider:

```sh
mise run build
```

Run unit tests:

```sh
mise run test
```

Run CCP acceptance tests (requires a real netcup account and environment variables):

```sh
NETCUP_CUSTOMER_NUMBER=... \
NETCUP_API_KEY=... \
NETCUP_API_PASSWORD=... \
mise run testacc
```

Run SCP tests (read-only data sources):

```sh
NETCUP_SCP_ACCESS_TOKEN=... \
mise run testacc ./internal/provider/scp/...
```

Generate Terraform Registry docs:

```sh
mise run docs
```

Releases are built by the `goreleaser/goreleaser-action` in
`.github/workflows/release.yml`.

## License

[MIT](./LICENSE)
