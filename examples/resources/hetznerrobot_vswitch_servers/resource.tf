resource "hetznerrobot_vswitch_servers" "main" {
  vlan_id = 4000
  servers = ["1234567"]
}
