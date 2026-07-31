resource "netcup_scp_user_firewall_policy" "example" {
  user_id = 1234

  name        = "allow-ssh-and-https"
  description = "Managed by Terraform"

  rules = [
    {
      action            = "ACCEPT"
      direction         = "INGRESS"
      protocol          = "TCP"
      destination_ports = "22"
      sources           = ["0.0.0.0/0"]
    },
    {
      action            = "ACCEPT"
      direction         = "INGRESS"
      protocol          = "TCP"
      destination_ports = "443"
      sources           = ["0.0.0.0/0"]
    },
  ]
}
