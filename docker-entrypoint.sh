#!/bin/sh

set -e

chroot-git clone /void-packages-origin /hostrepo
ln -s /hostrepo /opt/voidlinux/nbuild/void-packages
chroot-git -C /hostrepo remote set-url origin https://github.com/void-linux/void-packages.git

cat <<! >/hostrepo/etc/conf
XBPS_CHROOT_CMD=ethereal
XBPS_ALLOW_CHROOT_BREAKOUT=yes
!

exec /opt/voidlinux/nbuild/nbuild
