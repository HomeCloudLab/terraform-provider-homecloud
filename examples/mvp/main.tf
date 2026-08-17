terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

# Credentials: HC_ACCESS_KEY_ID + HC_SECRET_ACCESS_KEY (user-bound Access Key).
# Do not put secrets in this file. Override name_suffix if the default names exist.
provider "homecloud" {
  apex = "holab.abrdns.com"
}

variable "name_suffix" {
  type        = string
  default     = "demo"
  description = "Suffix so queue/bucket names stay unique in the account."
}

data "homecloud_account" "this" {}

resource "homecloud_mq_queue" "demo" {
  name = "tf-${var.name_suffix}-q"
}

resource "homecloud_so_bucket" "demo" {
  name = "tf-${var.name_suffix}-b"
}

output "account_number" {
  value = data.homecloud_account.this.account_number
}

output "queue_arn" {
  value = homecloud_mq_queue.demo.iam_arn
}

output "bucket_arn" {
  value = homecloud_so_bucket.demo.iam_arn
}
