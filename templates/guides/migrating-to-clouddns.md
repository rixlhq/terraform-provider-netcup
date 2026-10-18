---
page_title: "Migrating from Legacy DNS to CloudDNS - Terraform Provider for Netcup"
subcategory: "Guides"
description: |-
  Step-by-step guide for migrating existing netcup DNS configurations from the deprecated legacy CCP DNS API to the modern CloudDNS REST API.
---

# Migrating from Legacy DNS to CloudDNS

Netcup is deprecating the legacy Customer Control Panel (CCP) SOAP/JSON DNS API (`DomainWebservice`) in favor of the modern [netcup CloudDNS REST API](https://api.netcup.com).

The Terraform Provider for Netcup manages DNS exclusively via the modern **CloudDNS REST API** using `clouddns_api_key`. This guide outlines how to migrate your existing configurations without DNS downtime.

---

## 1. Prerequisites in the Netcup CCP

1. **Switch Domain Nameservers to CloudDNS**:
   - In the [netcup CCP](https://www.customercontrolpanel.de), navigate to **Domains**.
   - Select your domain, go to the **DNS** or **CloudDNS** tab, and activate CloudDNS.
   - Netcup automatically copies your existing zone records into CloudDNS.
2. **Generate a CloudDNS API Key**:
   - In the CCP, navigate to **Master Data > API > API Keys**.
   - Create an API key for the CloudDNS API (64-character bearer token).
   - Set the environment variable:
     ```shell
     export NETCUP_CLOUDDNS_API_KEY="your-64-character-clouddns-key"
     ```

---

## 2. Resource Mapping Reference

The resource and attribute mapping from the legacy DNS API to CloudDNS is straightforward:

| Legacy Resource (`netcup_dns_*`) | Legacy Attribute | CloudDNS Resource (`netcup_clouddns_*`) | CloudDNS Attribute | Notes |
| :--- | :--- | :--- | :--- | :--- |
| `netcup_dns_record` | `domain_name` | `netcup_clouddns_record` | `domain_id` | Use numeric domain ID or lookup from `data.netcup_clouddns_domains` |
| `netcup_dns_record` | `hostname` | `netcup_clouddns_record` | `name` | Relative record name (`@` for apex) |
| `netcup_dns_record` | `type` | `netcup_clouddns_record` | `type` | DNS type (e.g. `A`, `AAAA`, `CNAME`, `MX`, `TXT`) |
| `netcup_dns_record` | `destination` | `netcup_clouddns_record` | `rdata` | Record target/data in zone presentation format |
| `netcup_dns_record` | (implicit) | `netcup_clouddns_record` | `ttl` | Explicit TTL in seconds (e.g. `300`) |

---

## 3. Migration Example

### Before (Legacy)

```hcl
provider "netcup" {
  api_key         = var.netcup_api_key
  api_password    = var.netcup_api_password
  customer_number = var.netcup_customer_number
}

resource "netcup_dns_record" "web" {
  domain_name = "example.com"
  hostname    = "www"
  type        = "A"
  destination = "192.0.2.1"
}
```

### After (CloudDNS)

```hcl
provider "netcup" {
  clouddns_api_key = var.netcup_clouddns_api_key
}

data "netcup_clouddns_domains" "all" {}

locals {
  example_domain_id = one([
    for d in data.netcup_clouddns_domains.all.domains : d.id
    if d.fqdn == "example.com"
  ])
}

resource "netcup_clouddns_record" "web" {
  domain_id = local.example_domain_id
  name      = "www"
  type      = "A"
  rdata     = "192.0.2.1"
  ttl       = 300
}
```

---

## 4. Zero-Downtime State Migration

Because Netcup preserves your records when activating CloudDNS, you do not need to delete or recreate records. Use one of the following approaches:

### Option A: Declarative `import` block (Terraform >= 1.5)

Add an `import` block to your Terraform code using the human-friendly composite syntax (`domain_id/name/type/rdata`):

```hcl
import {
  to = netcup_clouddns_record.web
  id = "12345/www/A/192.0.2.1"
}
```

Run:
```shell
terraform state rm netcup_dns_record.web
terraform plan
terraform apply
```

### Option B: CLI Import

1. Remove the old legacy resource from state:
   ```shell
   terraform state rm netcup_dns_record.web
   ```
2. Import the existing record into the new `netcup_clouddns_record` resource:
   ```shell
   # Using composite syntax: domain_id/name/type/rdata
   terraform import netcup_clouddns_record.web '12345/www/A/192.0.2.1'

   # Or using UUID identifier from netcup_clouddns_records data source:
   terraform import netcup_clouddns_record.web '12345/59994952-e024-4871-b04d-546d66f25e73'
   ```
3. Run `terraform plan` to confirm no drift.
