resource "netcup_scp_server_action" "start" {
  server_id = 12345
  action    = "start"
}

resource "netcup_scp_server_action" "snapshot" {
  server_id = 12345
  action    = "snapshot_create"

  body = jsonencode({
    name   = "before-upgrade"
    online = true
  })
}

resource "netcup_scp_server_action" "revert" {
  server_id = 12345
  action    = "snapshot_revert"

  arguments = {
    snapshot_name = "before-upgrade"
  }

  triggers = {
    run = timestamp()
  }
}
