terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

provider "homecloud" {
  apex = "holab.abrdns.com"
}

variable "name_suffix" {
  type        = string
  default     = "demo"
  description = "Suffix so names stay unique in the account."
}

resource "homecloud_ssh_key" "ci" {
  name = "tf-${var.name_suffix}"
}

resource "homecloud_application" "api" {
  name     = "tf-${var.name_suffix}"
  slug     = "tf-${var.name_suffix}"
  template = "api-only"
}

# Provisions a VM. Skip unless you want Compute capacity.
# resource "homecloud_compute_machine" "web" {
#   name          = "tf-${var.name_suffix}"
#   machine_class = "basic"
#   image_id      = "ubuntu-24.04"
#   ssh_key_ids   = [homecloud_ssh_key.ci.id]
# }

output "ssh_public_key" {
  value = homecloud_ssh_key.ci.public_key
}

output "application_arn" {
  value = homecloud_application.api.iam_arn
}
