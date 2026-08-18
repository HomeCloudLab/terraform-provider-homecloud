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
  description = "Suffix so the secret name stays unique in the account."
}

resource "homecloud_secret" "demo" {
  name        = "tf-${var.name_suffix}-secret"
  description = "Terraform P1b example"
  values = {
    EXAMPLE_KEY = "not-a-real-secret"
  }
}

output "secret_arn" {
  value = homecloud_secret.demo.iam_arn
}
