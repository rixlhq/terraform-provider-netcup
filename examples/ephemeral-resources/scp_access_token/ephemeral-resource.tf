ephemeral "netcup_scp_access_token" "token" {
  refresh_token = var.netcup_scp_refresh_token
}

variable "netcup_scp_refresh_token" {
  type      = string
  sensitive = true
}
