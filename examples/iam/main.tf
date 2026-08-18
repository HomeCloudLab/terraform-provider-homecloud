terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

# IAM mutations need a User-bound Access Key whose console role is owner or admin
# (iam.manage). Developer keys 403. SA keys are not enabled on IAM routes.
provider "homecloud" {
  apex = "holab.abrdns.com"
}

variable "name_suffix" {
  type        = string
  default     = "demo"
  description = "Suffix so the policy/role names stay unique in the account."
}

data "homecloud_account" "this" {}

data "homecloud_iam_service_account" "functions" {
  name = "functions"
}

resource "homecloud_iam_policy" "mq" {
  name        = "tf-${var.name_suffix}-mq"
  description = "Terraform P2 example"
  document = jsonencode({
    Version = "2026-07-24"
    Statement = [{
      Effect   = "Allow"
      Action   = ["mq:*"]
      Resource = "arn:homecloud:mq::${data.homecloud_account.this.account_number}:queue/*"
    }]
  })
}

resource "homecloud_iam_role" "ci" {
  name        = "tf-${var.name_suffix}-ci"
  description = "Terraform P2 example role"
}

resource "homecloud_iam_policy_attachment" "functions_mq" {
  policy_arn     = homecloud_iam_policy.mq.arn
  principal_type = "service_account"
  principal_id   = data.homecloud_iam_service_account.functions.id
}

output "policy_arn" {
  value = homecloud_iam_policy.mq.arn
}

output "role_arn" {
  value = homecloud_iam_role.ci.arn
}

output "functions_sa_arn" {
  value = data.homecloud_iam_service_account.functions.arn
}
