data "netcup_scp_server_isoimages" "example" {
  server_id = 12345
}

output "available_iso_images" {
  value = [for iso in data.netcup_scp_server_isoimages.example.scp_server_isoimages : "${iso.name} (${iso.id})"]
}

locals {
  selected_iso = [
    for iso in data.netcup_scp_server_isoimages.example.scp_server_isoimages : iso
    if iso.name == "virtio-win.iso"
  ][0]
}
