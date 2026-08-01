resource "netcup_scp_server" "example" {
  server_id = 12345

  hostname = "srv.example.com"
  nickname = "My Server"

  autostart = true
  uefi      = false

  bootorder = ["HDD", "CDROM", "NETWORK"]

  os_optimization = "LINUX"
  keyboard_layout = "de"

  cpu_topology = {
    socket_count           = 1
    cores_per_socket_count = 2
  }
}
