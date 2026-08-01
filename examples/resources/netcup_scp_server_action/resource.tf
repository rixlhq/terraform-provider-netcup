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

resource "netcup_scp_server_action" "disk_driver" {
  server_id = 12345
  action    = "disk_driver_update"

  body = jsonencode({
    driver = "VIRTIO_SCSI"
  })
}

# Install an image from a public image flavour.
resource "netcup_scp_server_action" "image_setup" {
  server_id = 12345
  action    = "image_setup"

  body = jsonencode({
    imageFlavourId = 123
    diskName       = "sda"
    hostname       = "my-server"
    locale         = "en_US.UTF-8"
    timezone       = "Europe/Berlin"
  })
}

# Install a previously uploaded custom image.
resource "netcup_scp_server_action" "user_image_setup" {
  server_id = 12345
  action    = "user_image_setup"

  body = jsonencode({
    userImageName = "my-custom-image"
    diskName      = "sda"
  })
}

# Attach a public ISO image to a server.
resource "netcup_scp_server_action" "iso_attach" {
  server_id = 12345
  action    = "iso_attach"

  body = jsonencode({
    isoId                   = 456
    changeBootDeviceToCdrom = true
  })
}
