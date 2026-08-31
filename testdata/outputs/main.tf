variable "name" {
  type = string
}

output "bucket_name" {
  value = var.name
}
