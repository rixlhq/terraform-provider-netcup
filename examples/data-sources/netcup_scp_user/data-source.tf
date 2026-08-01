data "netcup_scp_user" "example" {
  user_id = 12345
}

output "user_email" {
  value = data.netcup_scp_user.example.email
}
