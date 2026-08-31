locals {
  name_prefix = "app-${var.region}"
  # dead local: computed once, referenced nowhere
  legacy_tags = { team = "platform" }
}

resource "aws_instance" "app" {
  count = var.instance_count
  tags = {
    Name = local.name_prefix
  }
}
