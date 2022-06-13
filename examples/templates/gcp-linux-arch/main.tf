terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "0.3.4"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 4.15"
    }
  }
}

provider "google" {
  zone    = "us-south1-a"
  project = "coder-dev-1"
}

data "google_compute_default_service_account" "default" {
}

data "coder_workspace" "me" {
}

data "google_compute_image" "arch_image" {
  family  = "arch"
  project = "arch-linux-gce"
}

resource "google_compute_disk" "root" {
  name  = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}-root"
  type  = "pd-ssd"
  image = data.google_compute_image.arch_image.self_link
  lifecycle {
    ignore_changes  = [image]
    prevent_destroy = false
  }
}

resource "coder_agent" "dev" {
  auth = "google-instance-identity"
  arch = "amd64"
  os   = "linux"
}

resource "google_compute_instance" "dev" {
  count        = data.coder_workspace.me.start_count
  name         = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  machine_type = "e2-medium"
  network_interface {
    network = "default"
    access_config {
      // Ephemeral public IP
    }
  }
  boot_disk {
    auto_delete = false
    source      = google_compute_disk.root.name
  }
  service_account {
    email  = data.google_compute_default_service_account.default.email
    scopes = ["cloud-platform"]
  }
  # The startup script runs as root with no $HOME environment set up, which can break workspace applications, so
  # instead of directly running the agent init script, setup the home directory, write the init script, and then execute
  # it.
  metadata_startup_script = <<EOMETA
#!/usr/bin/env sh
set -eux pipefail
mkdir /root || true
cat <<'EOCODER' > /root/coder_agent.sh
${coder_agent.dev.init_script}
EOCODER
chmod +x /root/coder_agent.sh

pacman-key --init
pacman-key --populate archlinux
pacman -S which --noconfirm

export HOME=/root
/root/coder_agent.sh

EOMETA
}
