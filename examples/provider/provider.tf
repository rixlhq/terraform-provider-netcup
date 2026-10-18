terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 2.0"
    }
  }
}

provider "netcup" {
  clouddns_api_key = var.netcup_clouddns_api_key # CCP Master Data > API > API Keys

  scp_access_token  = var.netcup_scp_access_token  # SCP OAuth / Bearer access token
  scp_refresh_token = var.netcup_scp_refresh_token # optional: for automatic token refresh
}
