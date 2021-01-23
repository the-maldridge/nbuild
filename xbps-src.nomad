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
cp -rv /hostrepo/hostdir/binpkgs/* /alloc/data/
EOF
        destination = "local/entrypoint"
        perms = "0755"
      }
    }

    task "mc" {
      driver = "docker"

      vault {
        policies = ["void-secrets-minio-nbuild"]
      }

      lifecycle {
        hook = "poststop"
      }

      config {
        image = "minio/mc:RELEASE.2021-01-05T05-03-58Z"
        entrypoint = ["/local/entrypoint"]
      }

      template {
        data = <<EOT
#!/bin/bash
{{- with service "minio" }}
{{- with $c := index . 0 }}
{{- with $v := secret "secret/minio/nbuild" }}
mc alias set void http://{{$c.Address}}:{{$c.Port}} {{$v.Data.access_key}} "{{$v.Data.secret_key}}"
{{- end }}
{{- end }}
{{- end }}
for f in /alloc/data/*.xbps ; do
        file=$(basename $f)
        f1=$${file/*-/}
        f2=$${f1/.xbps/}
        pkg=$${file%-*}
        arch=$${f2##*.}
        ver=$${f2%.*}
        mc cp $f void/packages/$arch/$file
done
EOT
        destination = "local/entrypoint"
        perms = "0755"
      }
    }
  }
}
