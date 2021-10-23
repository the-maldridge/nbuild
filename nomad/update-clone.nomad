job "void-packages" {
  type = "sysbatch"
  datacenters = ["minicluster"]

  group "git" {
    network {
      mode = "bridge"
    }

    volume "void-packages" {
      type = "host"
      source = "void-packages"
      read_only = false
    }

    task "update" {
      driver = "docker"

      config {
        image = "alpine/git:latest"
        args = ["clone", "https://github.com/void-linux/void-packages.git", "."]
      }

      resources {
        memory = 1000
      }

      volume_mount {
        volume = "void-packages"
        destination = "/git"
      }
    }
  }
}
