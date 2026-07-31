data "netcup_scp_user" "example" {
  user_id = 12345
}

data "netcup_scp_user_isos" "example" {
  user_id = data.netcup_scp_user.example.id
}

output "custom_isos" {
  value = [for iso in data.netcup_scp_user_isos.example.scp_user_isos : iso.key]
}
