resource "netcup_scp_user" "example" {
  user_id = 12345

  language = "en"
  time_zone = "Europe/Berlin"

  show_nickname     = true
  passwordless_mode = false
  secure_mode       = true
}
