resource "hetznerrobot_vswitch_servers" "main" {
  vlan    = 4000
  servers = ["1234567"]
}
