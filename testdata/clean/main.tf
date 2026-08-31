variable "region" {
  type = string
}

locals {
  prefix = "svc-${var.region}"
}

resource "aws_s3_bucket" "b" {
  bucket = local.prefix
}
