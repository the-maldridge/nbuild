job "void-packages" {
  type = "batch" // change to sysbatch after 1.2
  datacenters = ["dc1"]

  group "git" {

    network {
      mode = "bridge"
    }

    volume "void-packages" {
      type = "host"
      source = "void_packages"
      read_only = false
    }

    task "update" {
      driver = "docker"

      config {
        image = "alpine/git:latest"
        args = ["clone", "https://github.com/void-linux/void-packages.git", "."]
      }

      volume_mount {
        volume = "void-packages"
        destination = "/git"
      }
    }
  }
}
