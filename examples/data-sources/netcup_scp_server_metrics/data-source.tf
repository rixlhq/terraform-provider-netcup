data "netcup_scp_server_metrics" "cpu" {
  server_id = 12345
  metric    = "cpu"
  hours     = 1
}

locals {
  cpu_metrics = jsondecode(data.netcup_scp_server_metrics.cpu.json)
}
