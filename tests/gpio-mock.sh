#!/usr/bin/env bash

set -euo pipefail

gpio_sim_device=/sys/kernel/config/gpio-sim/matter-pi-gpio

if [ "${1:-}" = "teardown" ]; then
  if [ -d "$gpio_sim_device" ]; then
    echo 0 | sudo tee "$gpio_sim_device/live" >/dev/null
    sudo rmdir "$gpio_sim_device/bank0" "$gpio_sim_device"
  fi
  sudo modprobe -r gpio-sim
  exit 0
fi

sudo apt-get update
sudo apt-get install -y "linux-modules-extra-$(uname -r)"

echo "Kernel version: $(uname -r)"

sudo modprobe gpio-sim
sudo mkdir "$gpio_sim_device"
sudo mkdir "$gpio_sim_device/bank0"
echo 16 | sudo tee "$gpio_sim_device/bank0/num_lines" >/dev/null
echo 1 | sudo tee "$gpio_sim_device/live" >/dev/null

gpio_mock_chip=$(ls /dev/gpiochip* | sort -n | head -n 1)

echo "GPIO Mockup chip: $gpio_mock_chip"
