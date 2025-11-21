locals {
  instance_names = toset(
    flatten(
      values(
        {
          for name, count in var.instance_counts : name => [
            for i in range(0, count) : format("%s-%d", name, i)
          ]
        }
      )
    )
  )
}

resource "incus_profile" "minici_worker" {
  name        = "worker"
  description = "Worker virtual machine"
  project     = var.project_name

  config = {
    "limits.cpu"    = 2
    "limits.memory" = "2GiB"
    "user.user-data" = templatefile(
      "${path.module}/cloud-init/worker.yml",
      { admin_public_ssh_key = file("${path.module}/.ssh_keys/admin.pub") },
    )
  }

  device {
    name = "root"
    type = "disk"
    properties = {
      path = "/"
      pool = var.storage_pool_name
    }
  }

  device {
    name = "eth0"
    type = "nic"
    properties = {
      network = var.network_name
    }
  }
}

resource "incus_instance" "minici_workers" {
  for_each = local.instance_names
  name     = each.key
  type     = "virtual-machine"
  image    = var.instance_image
  profiles = [incus_profile.minici_worker.name]
}
