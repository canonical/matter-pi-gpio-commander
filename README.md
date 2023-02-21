# Matter Pi GPIO Commander
This app is a Matter lighting device which can be used to control the Raspberry Pi's GPIO. This can be used to control an LED or any other device.

## Install

```bash
sudo snap install matter-pi-gpio-commander
```

For installing on a classic Ubuntu or any other Linux distro with snap confinement but without the `gpio` interface:
```bash
sudo snap install matter-pi-gpio-commander --devmode
```
Refer to [GPIO.md](GPIO.md) for details.

### Configure
#### Set the pin
> **Warning**  
> The WiringPi pin numbering assignment differs from the physical pin and Raspberry Pi GPIO (BCM-GPIO).
> For example, on a Raspberry Pi 4B, the WiringPi pin 8 corresponds to physical pin 3 and GPIO 2.
> 
> For reference, visit https://pinout.xyz/pinout/wiringpi

```bash
sudo snap set matter-pi-gpio-commander wiringpi-pin=7
```

Make sure to also [grant the GPIO access](#GPIO).

#### Set CLI flags
The lighting app runs as a service without any default CLI flags. The snap allows passing flags to the service via the `args` snap option. 

For example, to set the `--wifi --passcode 1234` flags:
```
snap set matter-pi-gpio-commander args="--wifi --passcode 1234"
```

To see the list of all flags, run the `help` app:
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


### Grant access
The snap uses [interfaces](https://snapcraft.io/docs/interface-management) to allow access to external resources. Depending on the use case, you need to "connect" certain interfaces to grant the necessary access.
#### DNS-SD
The [avahi-control](https://snapcraft.io/docs/avahi-control-interface) is necessary to allow discovery of the application via DNS-SD:

```bash
sudo snap connect matter-pi-gpio-commander:avahi-control
```

> **Note**  
> To make DNS-SD discovery work, the host also needs to have a running avahi-daemon which can be installed with `sudo apt install avahi-daemon` or `snap install avahi`.

#### GPIO
The [`gpio`](https://snapcraft.io/docs/gpio-interface) interface provides slots for each GPIO channel. 

The slots can be listed using:
```bash
$ sudo snap interface gpio
name:    gpio
summary: allows access to specific GPIO pin
plugs:
  - matter-pi-gpio-commander
slots:
  - pi:bcm-gpio-0
  - pi:bcm-gpio-1
  - pi:bcm-gpio-10
  ...
```

> **Warning**  
> If no `slots` are listed, it means that there is no `gpio` interface available.
> In this case, you may skip the connection or may need to install in development mode.
> Refer to [GPIO.md](GPIO.md) for details.

The slots are not connected automatically. For example, to connect GPIO-4 (WiringPi pin 7 / physical pin 7):
```bash
sudo snap connect matter-pi-gpio-commander:gpio pi:bcm-gpio-4
```

Check the list of connections:
```
$ sudo snap connections
Interface        Plug                            Slot              Notes
gpio             matter-pi-gpio-commander:gpio   pi:bcm-gpio-4     manual
...
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
sudo chip-tool pairing ethernet 110 20202021 3840 192.168.1.111 5540
```

where:

-   `110` is the assigned node id
-   `20202021` is the pin code for the lighting app
-   `3840` is the discriminator id
-   `192.168.1.111` is the IP address of the host for the lighting app
-   `5540` the the port for the lighting app

Alternatively, to commission with discovery which works with DNS-SD:

```bash
sudo chip-tool pairing onnetwork 110 20202021
```

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

To use, install the snap and configure the WiringPi pin as explained above.
Then, run it via `sudo snap run matter-pi-gpio-commander.test-blink` snap command or directly:
```bash
sudo matter-pi-gpio-commander.test-blink
```

If you get the following error, it means that the GPIO access is not allowed. Refer to [GPIO.md](GPIO.md) for details.
```
sudo snap run matter-pi-gpio-commander.test-blink
wiringPiSetup: Unable to open /dev/mem or /dev/gpiomem: Permission denied.
  Aborting your program because if it can not access the GPIO
  hardware then it most certianly won't work
  Try running with sudo?
```
