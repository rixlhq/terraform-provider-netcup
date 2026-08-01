data "netcup_scp_user" "example" {
  user_id = 12345
}

data "netcup_scp_user_images" "example" {
  user_id = data.netcup_scp_user.example.id
}

output "custom_images" {
  value = [for img in data.netcup_scp_user_images.example.scp_user_images : img.key]
}
