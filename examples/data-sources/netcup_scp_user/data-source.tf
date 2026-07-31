data "netcup_scp_user" "example" {
  user_id = "me"
}

output "user_email" {
  value = data.netcup_scp_user.example.email
}
