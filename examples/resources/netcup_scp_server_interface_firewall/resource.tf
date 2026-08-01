resource "netcup_scp_server_interface_firewall" "example" {
  server_id = 12345
  mac       = "00:50:56:00:00:01"
  active    = true

  copied_policy_ids = [1, 2]
  user_policy_ids   = [10]
}
