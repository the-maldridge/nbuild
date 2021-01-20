job "update-void-packages" {
  type = "batch"
  namespace = "build"
  datacenters = ["VOID"]

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

      env {
        # ugly hack to get nomad to re-run the job with the same parameters
        SERIAL=1
      }

      volume_mount {
        volume = "void-packages"
        destination = "/git"
      }
    }
  }
}
