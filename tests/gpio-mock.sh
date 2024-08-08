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

kernel_major_minor=$(uname -r | cut -d'.' -f1-2)

echo "Kernel version: $(uname -r)"

# From investigations that's what is missing between the linux headers and
# the driver
commit=36aa129f22

# Get GPIO Mockup driver
wget https://raw.githubusercontent.com/torvalds/linux/$commit/drivers/gpio/gpio-mockup.c
wget https://raw.githubusercontent.com/torvalds/linux/$commit/drivers/gpio/gpiolib.h

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

