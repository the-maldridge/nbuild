job "xbps-src" {
  datacenters = ["VOID"]
  namespace = "build"
  type = "batch"

  parameterized {
    payload = "forbidden"
    meta_required = [
      "callback_done",
      "callback_fail",
      "host_arch",
      "package",
      "revision",
      "target_arch",
    ]
  }

  group "xbps-src" {
    reschedule {
      attempts = 0
      unlimited = false
    }
    restart {
      attempts = 0
    }
    network {
      mode = "bridge"
    }
    volume "void-packages" {
      type = "host"
      read_only = true
      source = "void-packages"
    }
    task "xbps-src" {
      driver = "docker"

      volume_mount {
        volume = "void-packages"
        destination = "/void-packages-origin"
        read_only = true
      }

      config {
        image = "voidlinux/masterdir-${NOMAD_META_HOST_ARCH}:20200607RC01"
        command = "/local/entrypoint"
      }

      env {
        GIT_REV = "${NOMAD_META_REVISION}"
        HOST = "${NOMAD_META_HOST_ARCH}"
        TARGET = "${NOMAD_META_TARGET_ARCH}"
        CALLBACK_FAIL = "${NOMAD_META_CALLBACK_FAIL}"
        CALLBACK_DONE = "${NOMAD_META_CALLBACK_DONE}"
      }

      template {
        data = file("./chroot.sh")
        destination = "local/entrypoint"
        perms = "0755"
      }
    }
  }
}
