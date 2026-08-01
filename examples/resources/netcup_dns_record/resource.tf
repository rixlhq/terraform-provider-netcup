resource "netcup_dns_record" "www" {
  zone        = "example.com"
  hostname    = "www"
  type        = "CNAME"
  destination = "example.com"
}

resource "netcup_dns_record" "mx" {
  zone        = "example.com"
  hostname    = "@"
  type        = "MX"
  destination = "mail.example.com"
  priority    = 10
}
