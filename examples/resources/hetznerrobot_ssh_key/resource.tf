resource "hetznerrobot_ssh_key" "example" {
  name = "my-ssh-key"
  data = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIexample user@host"
}
