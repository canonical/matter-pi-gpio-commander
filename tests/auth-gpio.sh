#!/usr/bin/env bash

# Unsquash the squashfs filesystem
unsquashfs -f -d ./squashfs-root matter-pi-gpio-commander_2.0.0_arm64.snap

# Add "/sys/devices/platform/gpio-mockup.*/gpiochip*/dev" to read permission list
sed -i 's|/sys/devices/platform/axi/\*.pcie/\*.gpio/gpiochip4/dev|/sys/devices/platform/axi/*.pcie/*.gpio/gpiochip4/dev\n      - /sys/devices/platform/gpio-mockup.*/gpiochip*/dev|' squashfs-root/meta/snap.yaml

# Recreate the squashfs filesystem with the prefix "mod_"
mksquashfs squashfs-root mod_matter-pi-gpio-commander_2.0.0_arm64.snap
