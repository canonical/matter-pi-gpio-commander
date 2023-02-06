# Matter Pi GPIO Commander
This app is a Matter lighting device which can be used to control LED using Raspberry Pi's GPIO pin.

The lighting device communicates with others over WiFi/Ethernet.

## Build
```bash
snapcraft -v
```
This will download >500MB and requires around 8GB of disk space. 

## Install
```bash
snap install --dangerous *.snap
```
For installing on a classic Ubuntu or any other Linux distro with snap confinement, add `--devmode`. Refer to [GPIO Access](GPIO.md) for details.

### Configure
> note
> TODO: The pin is currently hardcoded to Pin 7.

```bash
snap set matter-pi-gpio-commander gpio=17
```

### Connect interfaces
The [avahi-control](https://snapcraft.io/docs/avahi-control-interface) is necessary to allow discovery of the application via DNS-SD.
To make this work, the host also needs to have a running avahi-daemon which can be installed with `sudo apt install avahi-daemon` or `snap install avahi`.

```bash
snap connect matter-pi-gpio-commander:avahi-control
```

The `gpio` interface provides slots for each GPIO channel. The slots can be listed using:
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

The slots are not connected automatically. For example, to connect GPIO-7:
```bash
snap connect matter-pi-gpio-commander:gpio pi:bcm-gpio-7
```

Check the list of connections:
```
$ sudo snap connections
Interface        Plug                            Slot              Notes
gpio             matter-pi-gpio-commander:gpio   pi:bcm-gpio-7     manual
…


### Run
```bash
sudo snap start matter-pi-gpio-commander
sudo snap logs -f matter-pi-gpio-commander
```

## Control with Chip Tool

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

Assuming you have Ubuntu 22.04 and Python 3.10, install the following
dependencies:

### Dependencies
```
sudo apt install git gcc g++ libdbus-1-dev \
  ninja-build python3-venv python3-dev \
  python3-pip libgirepository1.0-dev libcairo2-dev
# maybe:
# sudo apt install pkg-config libssl-dev libglib2.0-dev libavahi-client-dev libreadline-dev
```

### Installation

Shallow clone the Connected Home IP project:
```bash
git clone https://github.com/project-chip/connectedhomeip.git --depth=1
cd ~/connectedhomeip/
scripts/checkout_submodules.py --shallow --platform linux
```

Build the Python/C libraries:
```bash
source ./scripts/activate.sh
./scripts/build_python_device.sh --chip_detail_logging true
```

Activate the Python env and install the dependencies inside it:

```bash
source ./out/python_env/bin/activate
pip install -r build/requirements.txt
```
