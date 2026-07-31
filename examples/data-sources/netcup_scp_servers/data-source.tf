data "netcup_scp_servers" "example" {
  limit = 10
}

output "server_names" {
  value = [for s in data.netcup_scp_servers.example.scp_servers : s.name]
}
