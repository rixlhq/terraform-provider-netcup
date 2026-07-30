terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 0.1"
    }
  }
}

provider "netcup" {
  api_key         = var.netcup_api_key
  api_password    = var.netcup_api_password
  customer_number = var.netcup_customer_number
}
