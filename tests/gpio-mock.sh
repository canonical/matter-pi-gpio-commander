#!/usr/bin/env bash

set -euo pipefail

gpio_sim_device=/sys/kernel/config/gpio-sim/matter-pi-gpio

# Removing the configfs directories only succeeds once the device is not live
# anymore, so a failure here would leave a half-created device behind and make
# the next setup fail on an already existing directory. The teardown therefore
# reports what it could not remove instead of failing silently.
teardown() {
  if [ -d "$gpio_sim_device" ]; then
    echo 0 | sudo tee "$gpio_sim_device/live" >/dev/null || true
    sudo rmdir "$gpio_sim_device/bank0" 2>/dev/null || true
    sudo rmdir "$gpio_sim_device" 2>/dev/null || true
  fi

  if [ -d "$gpio_sim_device" ]; then
    echo "Warning: could not remove $gpio_sim_device" >&2
  fi

  sudo modprobe -r gpio-sim >/dev/null 2>&1 || true
}

# The chip name and the platform device name are owned by configfs,
# so the simulator state is always queried from there instead of
# being cached in a file.
chip_name() {
  if [ ! -e "$gpio_sim_device/bank0/chip_name" ]; then
    echo "The GPIO simulator is not set up" >&2
    exit 1
  fi
  cat "$gpio_sim_device/bank0/chip_name"
}

sysfs_path() {
  local name dev
  name=$(chip_name) || exit 1
  dev=$(cat "$gpio_sim_device/dev_name") || exit 1
  echo "/sys/devices/platform/$dev/$name"
}

case "${1:-}" in
  teardown)
    teardown
    exit 0
    ;;
  chip)
    chip_name | sed 's/^gpiochip//'
    exit 0
    ;;
  state)
    line_offset="${2:?GPIO line offset is required}"
    state_path=$(sysfs_path) || exit 1
    value_path="$state_path/sim_gpio${line_offset}/value"

    if [ ! -e "$value_path" ]; then
      echo "The GPIO simulator has no line $line_offset: $value_path does not exist" >&2
      exit 1
    fi

    value=$(cat "$value_path")

    case "$value" in
      0) echo "low" ;;
      1) echo "high" ;;
      *)
        echo "Unexpected GPIO value: $value" >&2
        exit 1
        ;;
    esac
    exit 0
    ;;
esac

teardown

if ! modinfo gpio-sim >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y "linux-modules-extra-$(uname -r)"
fi

echo "Kernel version: $(uname -r)"

sudo modprobe configfs
if ! mountpoint -q /sys/kernel/config; then
  sudo mount -t configfs configfs /sys/kernel/config
fi

sudo modprobe gpio-sim
sudo mkdir -p "$gpio_sim_device"
sudo mkdir -p "$gpio_sim_device/bank0"
echo 16 | sudo tee "$gpio_sim_device/bank0/num_lines" >/dev/null
echo 1 | sudo tee "$gpio_sim_device/live" >/dev/null

actual_gpio_mock_chip=$(cat "$gpio_sim_device/bank0/chip_name")
if [ -z "$actual_gpio_mock_chip" ]; then
  echo "Failed to find the gpio-sim chip" >&2
  exit 1
fi

for _ in $(seq 1 50); do
  [ -e "/dev/$actual_gpio_mock_chip" ] && break
  sleep 0.1
done
if [ ! -e "/dev/$actual_gpio_mock_chip" ]; then
  echo "Device node /dev/$actual_gpio_mock_chip was not created" >&2
  exit 1
fi

gpio_chip_number=${actual_gpio_mock_chip#gpiochip}
if [ "$gpio_chip_number" != "0" ] && [ "$gpio_chip_number" != "4" ]; then
  echo "gpio-sim created /dev/$actual_gpio_mock_chip, but the snap only permits" \
    "gpiochip0 or gpiochip4. Simulated GPIO therefore requires a machine without" \
    "real GPIO chips, such as a CI runner." >&2
  exit 1
fi

gpio_sysfs_path=$(sysfs_path)
if [ ! -e "$gpio_sysfs_path/sim_gpio0/value" ]; then
  echo "GPIO simulator state path was not created: $gpio_sysfs_path" >&2
  exit 1
fi

echo "GPIO simulator chip: /dev/$actual_gpio_mock_chip"
echo "GPIO simulator sysfs path: $gpio_sysfs_path"
