data "netcup_scp_server" "example" {
  server_id = 12345
}

output "server_name" {
  value = data.netcup_scp_server.example.name
}

output "server_ipv4" {
  value = [for ip in data.netcup_scp_server.example.ipv4addresses : ip.ip]
}
