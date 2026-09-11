#!/usr/bin/env bash

set -euo pipefail

gpio_sim_device=/sys/kernel/config/gpio-sim/matter-pi-gpio
state_file=/tmp/matter-pi-gpio-sim-chip
sysfs_path_file=/tmp/matter-pi-gpio-sim-sysfs

teardown() {
  if [ -d "$gpio_sim_device" ]; then
    echo 0 | sudo tee "$gpio_sim_device/live" >/dev/null
    sudo rmdir "$gpio_sim_device/bank0" "$gpio_sim_device"
  fi

  sudo modprobe -r gpio-sim >/dev/null 2>&1 || true
  rm -f "$state_file" "$sysfs_path_file"
}

if [ "${1:-}" = "teardown" ]; then
  teardown
  exit 0
fi

if [ "${1:-}" = "state" ]; then
  line_offset="${2:?GPIO line offset is required}"
  gpio_sysfs_path=$(cat "$sysfs_path_file")
  value=$(cat "$gpio_sysfs_path/sim_gpio${line_offset}/value")

  case "$value" in
    0) echo "low" ;;
    1) echo "high" ;;
    *)
      echo "Unexpected GPIO value: $value" >&2
      exit 1
      ;;
  esac
  exit
fi

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
sudo mkdir "$gpio_sim_device"
sudo mkdir "$gpio_sim_device/bank0"
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

actual_gpio_chip_number=${actual_gpio_mock_chip#gpiochip}
gpio_mock_chip=$actual_gpio_mock_chip
gpio_chip_number=$actual_gpio_chip_number
if [ "$gpio_chip_number" != "0" ] && [ "$gpio_chip_number" != "4" ]; then
  echo "gpio-sim created /dev/$gpio_mock_chip; the snap only permits gpiochip0 or gpiochip4" >&2
  exit 1
fi

gpio_class_path="/sys/class/gpio/${actual_gpio_mock_chip/gpiochip/chip}"
if [ ! -e "$gpio_class_path" ]; then
  gpio_class_path="/sys/class/gpio/$actual_gpio_mock_chip"
fi
gpio_sysfs_path=$(readlink -f "$gpio_class_path")
echo "$gpio_chip_number" > "$state_file"
echo "$gpio_sysfs_path" > "$sysfs_path_file"

echo "GPIO simulator chip: /dev/$gpio_mock_chip"
echo "GPIO simulator sysfs path: $gpio_sysfs_path"
