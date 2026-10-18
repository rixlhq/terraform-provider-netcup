data "netcup_clouddns_records" "example" {
  domain_id = 12345
}

output "clouddns_records" {
  value = data.netcup_clouddns_records.example.records
}
