#!/usr/bin/env bash

set -euo pipefail

gpio_sim_device=/sys/kernel/config/gpio-sim/matter-pi-gpio
state_file=/tmp/matter-pi-gpio-sim-chip

teardown() {
  if [ -d "$gpio_sim_device" ]; then
    echo 0 | sudo tee "$gpio_sim_device/live" >/dev/null
    sudo rmdir "$gpio_sim_device/bank0" "$gpio_sim_device"
  fi

  sudo modprobe -r gpio-sim >/dev/null 2>&1 || true
  rm -f "$state_file"
}

if [ "${1:-}" = "teardown" ]; then
  teardown
  exit 0
fi

if [ "${1:-}" = "state" ]; then
  chip_number="${2:?GPIO chip number is required}"
  line_offset="${3:?GPIO line offset is required}"

  if ! mountpoint -q /sys/kernel/debug; then
    sudo mount -t debugfs debugfs /sys/kernel/debug
  fi

  sudo awk -v chip="gpiochip${chip_number}:" -v offset="$line_offset" '
    $1 == chip {
      in_chip = 1
      base = 0
      if ($2 == "GPIOs") {
        split($3, range, "-")
        base = range[1]
      }
      gpio = "gpio-" (base + offset)
      next
    }
    in_chip && /^gpiochip/ { exit 1 }
    in_chip && $1 == gpio && $0 ~ /(output-high|out hi)(,| |$)/ {
      print "high"; found = 1; exit
    }
    in_chip && $1 == gpio && $0 ~ /(output-low|out lo)(,| |$)/ {
      print "low"; found = 1; exit
    }
    END { if (!found) exit 1 }
  ' /sys/kernel/debug/gpio
  exit 0
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

if ! mountpoint -q /sys/kernel/debug; then
  sudo mount -t debugfs debugfs /sys/kernel/debug
fi

echo "GPIO simulator chip: /dev/$gpio_mock_chip"
echo "GPIO simulator sysfs path: $gpio_sysfs_path"
