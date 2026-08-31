variable "region" {
  type    = string
  default = "eu-central-1"
}

variable "instance_count" {
  type = number
}

# instance_type was left behind after a refactor and nothing wires it anymore.
variable "instance_type" {
  type    = string
  default = "t3.micro"
}
