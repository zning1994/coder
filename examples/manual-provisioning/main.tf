terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "0.3.4"
    }
  }
}

data "coder_workspace" "me" {
}

resource "coder_agent" "dev" {
  auth = "token"
  arch = "amd64"
  os   = "linux"
}

output "coder_token" {
  value = coder_agent.dev.token
}
