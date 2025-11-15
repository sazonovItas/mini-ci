variable "project_name" {
  type    = string
  default = "default"
}

variable "storage_pool_name" {
  type    = string
  default = "default"
}

variable "network_name" {
  type    = string
  default = "incusbr0"
}

variable "instance_counts" {
  type = map(number)
  default = {
    "minici-worker" = 1
  }
}

variable "instance_image" {
  type    = string
  default = "images:debian/12/cloud"
}
