terraform {
  required_providers {
    netcup = {
      source  = "rixlhq/netcup"
      version = "~> 0.1"
    }
  }
}

resource "netcup_scp_rdns" "example" {
  ip_version = provider::netcup::ip_version("1.2.3.4")
  ip         = "1.2.3.4"
  rdns       = "host.example.com"
}
