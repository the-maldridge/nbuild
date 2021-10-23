job "xbps-src" {
  datacenters = ["minicluster"]
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

      meta {
        nbuild_host = "${NOMAD_META_HOST_ARCH}"
        nbuild_target = "${NOMAD_META_TARGET_ARCH}"
        nbuild_package = "${NOMAD_META_PACKAGE}"
      }

      volume_mount {
        volume = "void-packages"
        destination = "/void-packages-origin"
        read_only = true
      }

      config {
        image = "ghcr.io/void-linux/xbps-src-masterdir:v20211022RC01-${NOMAD_META_HOST_ARCH}"
        command = "/usr/bin/sh"
        args = ["-x", "/local/entrypoint"]
      }

      resources {
        memory = 2000
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
        perms = "755"
      }
    }
  }
}
