data "netcup_dns_records" "example" {
  domain_name = "example.com"
}

output "records" {
  value = data.netcup_dns_records.example.records
}
