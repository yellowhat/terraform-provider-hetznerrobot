resource "hetznerrobot_failover" "test" {
  ip               = "1.2.3.4"
  active_server_ip = "5.6.7.8"
}
