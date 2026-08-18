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

resource "homecloud_function" "hello" {
  name    = "tf-${var.name_suffix}-fn"
  handler = "main.handler"
}

resource "homecloud_function_url" "hello" {
  function_name = homecloud_function.hello.name
}

resource "homecloud_ir_repository" "app" {
  name = "tf-${var.name_suffix}"
}

resource "homecloud_domain" "site" {
  hostname = "tf-${var.name_suffix}.example.com"
}

output "function_arn" {
  value = homecloud_function.hello.iam_arn
}

output "function_url" {
  value = homecloud_function_url.hello.function_url
}

output "repository_ref" {
  value = homecloud_ir_repository.app.image_ref_prefix
}
