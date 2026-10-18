# Terraform Provider for Netcup

[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-rixlhq%2Fnetcup-844FBA?logo=terraform&logoColor=white)](https://registry.terraform.io/providers/rixlhq/netcup/latest)
[![Release](https://img.shields.io/github/v/release/rixlhq/terraform-provider-netcup?sort=semver)](https://github.com/rixlhq/terraform-provider-netcup/releases)
[![CI](https://github.com/rixlhq/terraform-provider-netcup/actions/workflows/ci.yml/badge.svg)](https://github.com/rixlhq/terraform-provider-netcup/actions/workflows/ci.yml)
[![License: MPL-2.0](https://img.shields.io/github/license/rixlhq/terraform-provider-netcup)](LICENSE)

A [Terraform](https://www.terraform.io) provider for managing infrastructure at [netcup](https://www.netcup.com) via official netcup REST APIs:

- **[CloudDNS REST API](https://www.netcup.com/en/helpcenter/documentation/domain/cloud-dns)** (Bearer API) — modern replacement for the legacy Customer Control Panel (CCP) DomainWebservice DNS API; manage DNS records and query domain metadata with atomic changesets.
- **[Server Control Panel (SCP) REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api)** (OpenAPI / OAuth 2.0) — manage virtual servers (VPS / Root Server), reverse DNS (rDNS IPv4 & IPv6), network interfaces, VLANs, failover IPs, firewall policies, snapshots, custom disk images, and ISOs.

Built with the modern **Terraform Plugin Framework** (protocol version 6), supporting ephemeral token exchange (`netcup_scp_access_token`) and provider functions (`provider::netcup::ip_version`).

> [!TIP]
> **Looking for Customer Control Panel (CCP) DNS API support?**
> Netcup has transitioned from the legacy CCP SOAP/JSON API (`DomainWebservice`) to the modern, high-performance **[CloudDNS REST API](https://www.netcup.com/en/helpcenter/documentation/domain/cloud-dns)**. This provider integrates directly with the CloudDNS REST API. If you have existing DNS records created under the CCP, see the [CloudDNS Migration Guide](docs/guides/migrating-to-clouddns.md) for step-by-step instructions on activating CloudDNS and importing records into Terraform with zero downtime.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 (>= 1.10 for ephemeral resources, >= 1.8 for provider-defined functions)
- [Go](https://go.dev/doc/install) >= 1.26
- [mise](https://mise.jdx.dev/) (recommended for running tasks and installing tools)

## Overview of Resources & Data Sources

| Category | API | Resources | Data Sources |
| :--- | :--- | :--- | :--- |
| **CloudDNS** | [CloudDNS REST API](https://www.netcup.com/en/helpcenter/documentation/domain/cloud-dns) | `netcup_clouddns_record` | `netcup_clouddns_domains`, `netcup_clouddns_records` |
| **Servers** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_server`, `netcup_scp_server_action` | `netcup_scp_server`, `netcup_scp_servers`, `netcup_scp_server_metrics`, `netcup_scp_server_logs` |
| **Networking & rDNS** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_rdns`, `netcup_scp_server_interface`, `netcup_scp_failover_ip_v4`, `netcup_scp_failover_ip_v6`, `netcup_scp_user_vlan` | `netcup_scp_rdns_ipv4`, `netcup_scp_rdns_ipv6`, `netcup_scp_server_interface`, `netcup_scp_server_interfaces`, `netcup_scp_user_failoverips_v4`, `netcup_scp_user_failoverips_v6`, `netcup_scp_user_vlan`, `netcup_scp_user_vlans`, `netcup_scp_vlan` |
| **Firewall** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_user_firewall_policy`, `netcup_scp_server_interface_firewall` | `netcup_scp_user_firewall_policy`, `netcup_scp_user_firewall_policies` |
| **Storage & Snapshots** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_server_snapshot` | `netcup_scp_server_snapshot`, `netcup_scp_server_snapshots`, `netcup_scp_server_disk`, `netcup_scp_server_disks`, `netcup_scp_server_disk_supported_drivers`, `netcup_scp_server_imageflavours`, `netcup_scp_server_iso`, `netcup_scp_server_isoimages`, `netcup_scp_user_image`, `netcup_scp_user_image_part`, `netcup_scp_user_images`, `netcup_scp_user_iso`, `netcup_scp_user_iso_part`, `netcup_scp_user_isos` |
| **User & Access** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_user`, `netcup_scp_user_ssh_key`, `netcup_scp_access_token` (ephemeral) | `netcup_scp_user`, `netcup_scp_user_ssh_keys`, `netcup_scp_user_logs` |
| **Tasks & Status** | [SCP REST API](https://www.netcup.com/en/helpcenter/documentation/server/rest-api) | `netcup_scp_task_action` | `netcup_scp_task`, `netcup_scp_tasks`, `netcup_scp_maintenance`, `netcup_scp_server_guest_agent`, `netcup_scp_server_guest_agent_status`, `netcup_scp_server_gpu_driver`, `netcup_scp_server_rescuesystem` |

## Provider Configuration

Configure credentials for CloudDNS and/or SCP REST APIs:

```hcl
terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 2.0"
    }
  }
}

provider "netcup" {
  clouddns_api_key = var.netcup_clouddns_api_key # CCP Master Data > API > API Keys

  scp_access_token  = var.netcup_scp_access_token  # SCP Bearer access token
  scp_refresh_token = var.netcup_scp_refresh_token # optional: for automatic token refresh
}
```

- **CloudDNS API keys** are generated in **CCP > Master Data > API > API Keys**.
- **SCP tokens** are generated via the netcup SCP OAuth 2.0 device-code flow.

## CloudDNS Resources and Data Sources

`netcup_clouddns_record` manages individual DNS records with atomic, strict changesets:

```hcl
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

Import existing records with either composite syntax (`domain_id/name/type/rdata`) or internal UUID:

```shell
terraform import netcup_clouddns_record.www '12345/www/A/192.0.2.10'
```

## Protocol Version 6 Features

The provider is built on Terraform Plugin Framework protocol version 6 and exposes features that require newer Terraform CLI versions:

- **Provider-defined functions** (`Terraform >= 1.8`):
  `provider::netcup::ip_version(ip)` returns `ipv4` or `ipv6` for a given address.
- **Ephemeral resources** (`Terraform >= 1.10`):
  `netcup_scp_access_token` exchanges a refresh token for a short-lived access token that is never stored in Terraform state or plan files.

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
