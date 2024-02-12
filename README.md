# Matter Pi GPIO Commander
[![matter-pi-gpio-commander](https://snapcraft.io/matter-pi-gpio-commander/badge.svg)](https://snapcraft.io/matter-pi-gpio-commander)

This small application can turn your Raspberry Pi into a Matter lighting device. Once setup and commissioned, it allows control of a configured GPIO pin via Matter on/off commands. The GPIO output can be used switch an LED or another device via a relay.

The application is based on [CHIP's Linux Lighting App](https://github.com/project-chip/connectedhomeip/tree/master/examples/lighting-app/linux) example. It used the [WiringPi](https://github.com/WiringPi/WiringPi) library to control the GPIO of Raspberry Pi.

Usage instructions are available below and on the **[wiki](https://github.com/canonical/matter-pi-gpio-commander/wiki)**.

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

## Install

```bash
sudo snap install matter-pi-gpio-commander
```

### Configure
#### Set the pin

```bash
sudo snap set matter-pi-gpio-commander gpio=4
```

Make sure to also [grant the GPIO access](#GPIO).

#### Set CLI flags
By default, the lighting app runs as a service without any CLI flags.
The snap allows passing flags to the service via the `args` snap option. 
This is useful for overriding SDK defaults to customize the application behavior.

To see the list of all flags and SDK default, run the `help` app:
```
$ matter-pi-gpio-commander.help
Usage: /snap/matter-pi-gpio-commander/x3/bin/lighting-app [opti

GENERAL OPTIONS

  --ble-device <number>
       The device number for CHIPoBLE, without 'hci' prefix, can be found by hciconfig.

  --wifi
       Enable WiFi management via wpa_supplicant.

  --thread
       Enable Thread management via ot-agent.

  ...

```

For example, to set Passcode for commissioning:
```bash
sudo snap set matter-pi-gpio-commander args="--passcode 1234"
```

For enabling Thread management:
```bash
sudo snap set matter-pi-gpio-commander args="--thread"
```

> **Note**  
> For Thread management, the application needs to have access to the OpenThread Border Router (OTBR) agent via DBus.
> When using the [OTBR Snap], this can be achieved by installing the snap and granting the necessary rights; refer to [Thread](#Thread).

For setting multiple flags, concatenate the arguments and set them together:
```bash
sudo snap set matter-pi-gpio-commander args="--thread --ble-device 1"
```


### Grant access
The snap uses [interfaces](https://snapcraft.io/docs/interface-management) to allow access to external resources. Depending on the use case, you need to "connect" certain interfaces to grant the necessary access.

#### DNS-SD
The [avahi-control](https://snapcraft.io/docs/avahi-control-interface) is necessary to allow discovery of the application via DNS-SD:

```bash
sudo snap connect matter-pi-gpio-commander:avahi-control
```

> **Note**  
> To make DNS-SD discovery work, the host also needs to have a running avahi-daemon which can be installed with `sudo apt install avahi-daemon`.


> **Note**  
> On **Ubuntu Core**, the `avahi-control` interface is not provided by the system. Instead, it depends on the [Avahi snap](https://snapcraft.io/avahi).
> To use the interface from that snap, run:
> ```bash
> sudo snap connect matter-pi-gpio-commander:avahi-control avahi:avahi-control
> ```

#### GPIO
The gpio access is granted using the [`custom-device`](https://snapcraft.io/docs/custom-device-interface), which declares a slot to expose the `/dev/gpiochip*` device and also a plug to self connect.
This interface is auto connected when installing the snap from the Snap Store.

For manual connection:
```bash
sudo snap connect matter-pi-gpio-commander:custom-gpio matter-pi-gpio-commander:custom-gpio-dev 
```

#### BLE
To allow the device to advertise itself over Bluetooth Low Energy:
```bash
sudo snap connect matter-pi-gpio-commander:bluez
```

> **Note**  
> BLE advertisement depends on BlueZ which can be installed with `sudo apt install bluez`.

> **Note**  
> On **Ubuntu Core**, the `bluez` interface is not provided by the system. 
> The interface can instead be consumed from the [BlueZ snap](https://snapcraft.io/bluez):
> ```bash
> sudo snap connect matter-pi-gpio-commander:bluez bluez:service
> ```


#### Thread 
To allow communication with the [OTBR Snap] for Thread management, connect the following interface:

```bash
sudo snap connect matter-pi-gpio-commander:otbr-dbus-wpan0 \
                  openthread-border-router:dbus-wpan0
```

### Run
```bash
sudo snap start matter-pi-gpio-commander
```
Add `--enable` to make the service automatically start at boot. 

Query and follow the logs:
```
sudo snap logs -n 100 -f matter-pi-gpio-commander
```

## Control with Chip Tool
For the following examples, we use the [Chip Tool snap](https://snapcraft.io/chip-tool) to commission and control the lighting app.
### Commissioning

```bash
sudo snap connect chip-tool:avahi-observe
sudo chip-tool pairing onnetwork 110 20202021
```

where:

-   `110` is the assigned node id
-   `20202021` is the default passcode (pin code) for the lighting app

### Command

Switching on/off:

```bash
sudo chip-tool onoff toggle 110 1 # toggle is stateless and recommended
sudo chip-tool onoff on 110 1
sudo chip-tool onoff off 110 1
```

where:

-   `onoff` is the matter cluster name
-   `on`/`off`/`toggle` is the command name. The `toggle` command is RECOMMENDED
    because it is stateless. The lighting app does not synchronize the actual state of
    devices.
-   `110` is the node id of the lighting app assigned during the commissioning
-   `1` is the endpoint of the configured device

## Development
Build:
```bash
snapcraft -v
```
This will download >500MB and requires around 8GB of disk space. 

To build for other architectures, customize the `architectures` field inside the snapcraft.yaml and use snapcraft's [Remote build](https://snapcraft.io/docs/remote-build).

Install it as described in the [install](#install) section by replacing `matter-pi-gpio-commander` with the locally built snap file name and setting `--dangerous` flag.

## Test Blink
This project includes an app to quickly verify the chosen pin and snap GPIO access control without using a Matter Controller.
The app will toggle the output voltage of the pin to high/low periodically.

To use, install the snap and configure the GPIO as explained above.
Then, run it via `sudo snap run matter-pi-gpio-commander.test-blink` snap command or directly:
```bash
sudo matter-pi-gpio-commander.test-blink
```

<!-- References -->
[OTBR Snap]: https://snapcraft.io/openthread-border-router
