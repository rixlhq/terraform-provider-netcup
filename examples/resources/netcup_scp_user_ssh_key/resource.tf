resource "netcup_scp_user_ssh_key" "example" {
  user_id = 12345
  name    = "my-key"
  key     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIhz2GK/XCu1krZ6B/jWhdd1Rzie/OXtmYPTQG5jmy7 example"
}
