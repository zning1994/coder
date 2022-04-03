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

// variable "image" {
//     description = <<EOF
// Google machien 
// EOF
//     sensitive = true
// }

// variable "service_account" {
//   description = <<EOF
// Coder requires a Google Cloud Service Account to provision workspaces.

// 1. Create a service account:
//    https://console.cloud.google.com/projectselector/iam-admin/serviceaccounts/create
// 2. Add the roles:
//    - Compute Admin
//    - Service Account User
// 3. Click on the created key, and navigate to the "Keys" tab.
// 4. Click "Add key", then "Create new key".
// 5. Generate a JSON private key, and paste the contents below.
// EOF
//   sensitive   = true
// }

variable "zone" {
  description = "What region should your workspace live in?"
  default     = "us-central1-a"
  validation {
    condition     = contains(["northamerica-northeast1-a", "us-central1-a", "us-west2-c", "europe-west4-b", "southamerica-east1-a"], var.zone)
    error_message = "Invalid zone!"
  }
}

provider "google" {
  zone        = var.zone
  project     = "coder-devrel"
}

// .tfvars is a solution
// We can still display parameters in the UI, but that's just not how user-specific
// parameters will work.
// 
// That way if a parameter is set poorly by a user, a delete can still occur.


// No builds for a project!
// When you create a workspace from an empty project,

data "coder_workspace" "me" {
}

data "coder_agent_script" "dev" {
  auth = "google-instance-identity"
  arch = "amd64"
  os   = "linux"
}

data "google_compute_default_service_account" "default" {
}

resource "google_compute_disk" "home" {
  name  = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}-home"
  type  = "pd-ssd"
  zone  = var.zone
  size = 20
  lifecycle {
    ignore_changes = [image]
  }
}

resource "google_compute_instance" "dev" {
  zone         = var.zone
  count        = data.coder_workspace.me.transition == "start" ? 1 : 0
  name         = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  machine_type = "c2d-highcpu-16"
  network_interface {
    network = "default"
    access_config {
      // Ephemeral public IP
    }
  }
  scratch_disk {
    interface = "SCSI"
  }
  boot_disk {
    initialize_params {
      image = "packer-1648759576"
      type = "pd-ssd"
    }
  }
  attached_disk {
      source = google_compute_disk.home.self_link
  }
  service_account {
    email  = data.google_compute_default_service_account.default.email
    scopes = ["cloud-platform"]
  }
  metadata = {
      hostname = data.coder_workspace.me.name
      startup-script = <<EOF
USER=${data.coder_workspace.me.owner}
useradd -m -s /bin/bash -G sudo -G docker $USER
sudo -E -u $USER sh -c '${data.coder_agent_script.dev.value}'
EOF
  }
}

resource "coder_agent" "dev" {
  count       = length(google_compute_instance.dev)
  instance_id = google_compute_instance.dev[0].instance_id
}
