data "netcup_clouddns_domains" "all" {}

output "clouddns_domains" {
  value = data.netcup_clouddns_domains.all.domains
}
