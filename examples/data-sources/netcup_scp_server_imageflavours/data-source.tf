data "netcup_scp_server_imageflavours" "example" {
  server_id = 12345
}

output "available_flavours" {
  value = [for f in data.netcup_scp_server_imageflavours.example.scp_server_imageflavours : "${f.name} (${f.id})"]
}

# Reference the first matching flavour by name in a server setup action.
locals {
  selected_flavour = [
    for f in data.netcup_scp_server_imageflavours.example.scp_server_imageflavours : f
    if f.name == "Debian 12"
  ][0]
}
