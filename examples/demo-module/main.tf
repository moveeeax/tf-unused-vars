variable "region" {
  type = string
}

# left over after a refactor — flagged by tf-unused-vars
variable "legacy_flag" {
  type    = bool
  default = false
}

locals {
  bucket = "logs-${var.region}"
}

resource "aws_s3_bucket" "logs" {
  bucket = local.bucket
}
