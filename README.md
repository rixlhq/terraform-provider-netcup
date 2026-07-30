# Terraform Provider for Netcup

A Terraform provider for managing DNS records and zones at [netcup](https://www.netcup.com) via the netcup Customer Control Panel (CCP) JSON API.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.26

## Provider Configuration

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
}
```

All credentials are generated in the netcup Customer Control Panel under **Master Data > API**.

## Resources

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

## Data Sources

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

## Limitations

- Netcup does not support per-record TTLs; TTL is set per zone. Use
  `netcup_dns_zone` to change it.
- Creating or deleting DNS zones is not supported by the CCP API; only records
  within an existing zone can be managed.

## Development

Build the provider:

```sh
go build ./...
```

Run unit tests:

```sh
go test ./...
```

Run acceptance tests (requires a real netcup account and environment variables):

```sh
NETCUP_CUSTOMER_NUMBER=... \
NETCUP_API_KEY=... \
NETCUP_API_PASSWORD=... \
TF_ACC=1 go test ./...
```

## License

[MIT](./LICENSE)
