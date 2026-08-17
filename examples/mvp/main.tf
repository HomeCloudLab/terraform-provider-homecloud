terraform {
  required_providers {
    homecloud = {
      source  = "homecloudlab/homecloud"
      version = ">= 0.1.0"
    }
  }
}

provider "homecloud" {
  apex = "holab.abrdns.com"
}

data "homecloud_account" "this" {}

resource "homecloud_mq_queue" "jobs" {
  name = "tf-jobs"
}

resource "homecloud_so_bucket" "assets" {
  name = "tf-assets"
}

output "account_number" {
  value = data.homecloud_account.this.account_number
}

output "queue_arn" {
  value = homecloud_mq_queue.jobs.iam_arn
}

output "bucket_arn" {
  value = homecloud_so_bucket.assets.iam_arn
}
