resource "netcup_dns_zone" "example" {
  domain_name = "example.com"
  ttl         = 3600
}
