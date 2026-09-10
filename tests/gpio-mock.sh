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

  gpio_base=$(
    sudo awk -v chip="gpiochip${chip_number}:" '
      $1 == chip {
        split($3, range, "-")
        print range[1]
        exit
      }
    ' /sys/kernel/debug/gpio
  )
  if [ -z "$gpio_base" ]; then
    echo "unknown"
    exit 1
  fi

  gpio_number=$((gpio_base + line_offset))
  sudo awk -v gpio="gpio-${gpio_number}" '
    $1 == gpio && $0 ~ / out hi( |$)/ { print "high"; found = 1; exit }
    $1 == gpio && $0 ~ / out lo( |$)/ { print "low"; found = 1; exit }
    END { if (!found) exit 1 }
  ' /sys/kernel/debug/gpio
  exit 0
fi

teardown

sudo apt-get update
sudo apt-get install -y "linux-modules-extra-$(uname -r)"

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

gpio_mock_chip=$(
  find /sys/class/gpio -maxdepth 1 -type l -name 'gpiochip*' -printf '%f\n' |
    while read -r chip; do
      if readlink -f "/sys/class/gpio/$chip" | grep -q '/gpio-sim\.'; then
        echo "$chip"
      fi
    done |
    sort -V |
    tail -n 1
)
if [ -z "$gpio_mock_chip" ]; then
  echo "Failed to find the gpio-sim chip" >&2
  exit 1
fi

gpio_chip_number=${gpio_mock_chip#gpiochip}
if [ "$gpio_chip_number" != "0" ] && [ "$gpio_chip_number" != "4" ]; then
  echo "gpio-sim created /dev/$gpio_mock_chip; the snap only permits gpiochip0 or gpiochip4" >&2
  exit 1
fi

gpio_sysfs_path=$(readlink -f "/sys/class/gpio/$gpio_mock_chip")
echo "$gpio_chip_number" > "$state_file"

if ! mountpoint -q /sys/kernel/debug; then
  sudo mount -t debugfs debugfs /sys/kernel/debug
fi

echo "GPIO simulator chip: /dev/$gpio_mock_chip"
echo "GPIO simulator sysfs path: $gpio_sysfs_path"
