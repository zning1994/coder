
terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 0.2"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 4.15"
    }
  }
}

provider "kubernetes" {
  # TODO: turn these into variables
  # generate these with the same steps for adding a k8s workspace 
  # provider in Coder v1
  host                   = ""
  cluster_ca_certificate = base64decode("")
  token                  = base64decode("")
}

resource "kubernetes_pod" "test" {

  count = data.coder_workspace.me.transition == "start" ? 1 : 0

  metadata {
    name = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  }


  spec {
    container {

      # TODO: turn these into variables
      image = "nginx:1.21.6"
      name  = "example"


      command = ["sh", "-c", coder_agent.dev[0].init_script]

      env {
        name  = "CODER_TOKEN"
        value = coder_agent.dev[0].token
      }

      env {
        name  = "environment"
        value = "test"
      }

      port {
        container_port = 80
      }

      volume_mount {
        mount_path = "/home/idk"
        name       = "home-directory"
      }

    }

    volume {
      name = "home-directory"
      persistent_volume_claim {
        claim_name = kubernetes_persistent_volume_claim.home-directory.metadata.0.name
      }
    }

  }
}

resource "kubernetes_persistent_volume_claim" "home-directory" {
  metadata {
    name = "coder-pvc-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  }
  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        # TODO: turn these into variables
        storage = "5Gi"
      }
    }
  }
}


data "coder_workspace" "me" {
}

resource "coder_agent" "dev" {
  count = data.coder_workspace.me.start_count
  auth  = "token"
  arch  = "amd64"
  os    = "linux"
}
