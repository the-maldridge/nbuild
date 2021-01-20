job "xbps-src" {
  datacenters = ["VOID"]
  namespace = "build"
  type = "batch"

  parameterized {
    payload = "forbidden"
    meta_required = ["package", "host_arch"]
    meta_optional = ["target_arch"]
  }

  group "xbps-src-pkg" {
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
    task "xbps-src-pkg" {
      driver = "docker"

      volume_mount {
        volume = "void-packages"
        destination = "/void-packages-origin"
        read_only = true
      }

      config {
        image = "voidlinux/masterdir-${NOMAD_META_HOST_ARCH}:20200607RC01"
        command = "/local/entrypoint"
        args = ["pkg", "${NOMAD_META_PACKAGE}"]
      }

      template {
        data = <<EOF
#!/bin/sh
set -e
xbps-install -Syu xbps
xbps-install -Syu git
git clone /void-packages-origin /hostrepo

cat <<! >/hostrepo/etc/conf
XBPS_CHROOT_CMD=ethereal
XBPS_ALLOW_CHROOT_BREAKOUT=yes
!
ln -s / /hostrepo/masterdir
./hostrepo/xbps-src "$@"
EOF
        destination = "local/entrypoint"
        perms = "755"
      }
    }
  }
}
