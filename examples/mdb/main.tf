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
  description = "Suffix so instance names stay unique in the account."
}

variable "db_password" {
  type      = string
  sensitive = true
  default   = "ChangeMe22"
}

resource "homecloud_mdb_instance" "app" {
  name           = "tf-${var.name_suffix}-pg"
  engine         = "postgresql"
  instance_class = "micro"
}

resource "homecloud_mdb_user" "ci" {
  instance_name = homecloud_mdb_instance.app.name
  username      = "ci"
  password      = var.db_password
  role          = "readwrite"
}

resource "homecloud_redis_instance" "cache" {
  name           = "tf-${var.name_suffix}-redis"
  instance_class = "micro"
}

output "mdb_arn" {
  value = homecloud_mdb_instance.app.iam_arn
}

output "mdb_endpoint" {
  value = homecloud_mdb_instance.app.internal_endpoint
}

output "redis_arn" {
  value = homecloud_redis_instance.cache.iam_arn
}
