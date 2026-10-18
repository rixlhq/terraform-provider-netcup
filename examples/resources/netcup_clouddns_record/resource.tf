# Supply NETCUP_CLOUDDNS_API_KEY from your secret manager.
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
