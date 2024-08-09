#!/usr/bin/env bash

if [ "$1" = "teardown" ]; then
  sudo rmmod gpio_mockup
  rm -rf gpio-mockup
  exit 0
fi

mkdir gpio-mockup
cd gpio-mockup

# Update and install dependencies
sudo apt-get update
sudo apt-get install -y linux-headers-$(uname -r)
sudo apt-get install -y build-essential flex bison make

echo "Kernel version: $(uname -r)"

. /etc/os-release

# Get GPIO Mockup driver
wget https://git.launchpad.net/~canonical-kernel/ubuntu/+source/linux-azure/+git/$UBUNTU_CODENAME/plain/drivers/gpio/gpio-mockup.c
wget https://git.launchpad.net/~canonical-kernel/ubuntu/+source/linux-azure/+git/$UBUNTU_CODENAME/plain/drivers/gpio/gpiolib.h

# Create Makefile
echo "
obj-m = gpio-mockup.o
KVERSION = \$(shell uname -r)
all:
	make -C /lib/modules/\$(KVERSION)/build M=\$(PWD) modules
clean:
	make -C /lib/modules/\$(KVERSION)/build M=\$(PWD) clean
" >Makefile

make -j$(nproc)

sudo insmod gpio-mockup.ko gpio_mockup_ranges=-1,16 gpio_mockup_named_lines

gpio_mock_chip=$(ls /dev/gpiochip* | sort -n | head -n 1)

echo "GPIO Mockup chip: $gpio_mock_chip"

cd ..
