# Matter Pi GPIO Commander
[![matter-pi-gpio-commander](https://snapcraft.io/matter-pi-gpio-commander/badge.svg)](https://snapcraft.io/matter-pi-gpio-commander)

This small application can turn your Raspberry Pi into a Matter lighting device. Once setup and commissioned, it allows control of a configured GPIO pin via Matter on/off commands.
The GPIO output can be used switch an LED or another device via a relay.

The application is based on [CHIP's Linux Lighting App](https://github.com/project-chip/connectedhomeip/tree/master/examples/lighting-app/linux) example.
It used the [character device](https://docs.kernel.org/userspace-api/gpio/chardev.html) API to control the GPIO of Raspberry Pi.

## Hardware Compatibility

This snap is expected to work on the following Raspberry Pi hardware:

- RPi 5 Model B Rev 1.x
- RPi 4 Model B Rev 1.x
- RPi 400 Rev 1.x
- RPi CM4 Rev 1.x
- RPi 3 Model B Rev 1.x
- RPi 3 Model B Plus Rev 1.x
- RPi 3 Model A Plus Rev 1.x
- RPi CM3 Rev 1.x
- RPi Zero 2W Rev 1.x

**Note:** If you have one of the listed hardware, and this snap doesn't work on it, please [open an issue](https://github.com/canonical/matter-pi-gpio-commander/issues/new).

## Usage
Please refer to
[this tutorial](https://canonical-matter.readthedocs-hosted.com/en/latest/tutorial/pi-gpio-commander/)
to install and configure the application.

## Development
Build:
```bash
snapcraft -v
```
This will download >500MB and requires around 8GB of disk space. 

To build for other architectures, customize the `architectures` field inside the snapcraft.yaml and use snapcraft's [Remote build](https://snapcraft.io/docs/remote-build).

Install:
```bash
sudo snap install --dangerous *.snap
```

Manually connect the following interface:
```bash
sudo snap connect matter-pi-gpio-commander:custom-gpio matter-pi-gpio-commander:custom-gpio-dev 
```

Continue by following the [usage](#usage) instructions.

## Test Blink
This project includes an app to quickly validate the GPIO configuration without using a Matter Controller.
The app will toggle the output voltage of the pin to high/low periodically.

To use, install the snap and configure the GPIO.
Then, run it via `sudo snap run matter-pi-gpio-commander.test-blink` snap command or directly:
```bash
sudo matter-pi-gpio-commander.test-blink
```

