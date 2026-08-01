resource "netcup_scp_server_snapshot" "example" {
  server_id       = 1234
  name            = "terraform-snapshot"
  description     = "Managed by Terraform"
  disk_name       = "vda"
  online_snapshot = true
}
