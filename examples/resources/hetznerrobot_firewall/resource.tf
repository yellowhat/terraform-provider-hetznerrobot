resource "hetznerrobot_firewall" "firewall" {
  server_id     = 1234567
  active        = true
  whitelist_hos = true

  rule {
    name     = "icmp"
    protocol = "icmp"
    action   = "accept"
  }

  rule {
    name     = "ssh"
    protocol = "tcp"
    dst_port = "22"
    action   = "accept"
  }

  rule {
    name   = "Deny others"
    action = "discard"
  }
}
