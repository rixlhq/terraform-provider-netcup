resource "netcup_scp_server_interface" "example" {
  server_id      = 12345
  vlan_id        = 100
  network_driver = "virtio"
}
