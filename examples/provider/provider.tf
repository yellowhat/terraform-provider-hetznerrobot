terraform {
  required_providers {
    hetznerrobot = {
      source  = "yellowhat/hetzner-robot"
      version = "1.0.0"
    }
  }
}

provider "hetznerrobot" {
  username = "yourUserNameFromRobot"
  password = "yourPasswordFromRobot"
}
