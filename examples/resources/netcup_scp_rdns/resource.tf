resource "netcup_scp_rdns" "example" {
  ip_version = "ipv4"
  ip         = "192.0.2.10"
  rdns       = "host.example.com"
}
