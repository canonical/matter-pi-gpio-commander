# Tapo Matter Bridge
This is year 2022 and TP-Link Tapo devices aren't yet Matter-ready.

This app is a Matter bridge which can be used to turn the Tapo L530E into a Matter device.

The bridge communicates with a single, pre-commissioned Tapo device over WiFi.
## Snap
### Build and install
```bash
snapcraft -v
snap install --dangerous ./matter-bridge-tapo-l530_0.1_amd64.snap
```
### Configure
```bash
snap set matter-bridge-tapo-l530 ip="tapo device ip"
snap set matter-bridge-tapo-l530 user="tapo user"
snap set matter-bridge-tapo-l530 password="tapo password"
```

### Connect interfaces
```bash
snap connect matter-bridge-tapo-l530:avahi-control
```

The [avahi-control](https://snapcraft.io/docs/avahi-control-interface) is necessary to allow discovery of the application via DNS-SD.
To make this work, the host also needs to have a running avahi-daemon which can be installed with `sudo apt install avahi-daemon` or `snap install avahi`.

### Run
```bash
sudo snap start matter-bridge-tapo-l530
sudo snap logs -f matter-bridge-tapo-l530
```

## Native

Assuming you have setup the Connected Home IP project for Python projects (see [Development](#development)) at `../connectedhomeip`:

### Activate the Python env
```bash
source ../connectedhomeip/out/python_env/bin/activate
```

### Run
```bash
IP="tapo device IP" USER="tapo user" PASSWORD="tapo password" python lighting.py
```

## Control with Chip Tool

### Commissioning

```bash
chip-tool pairing ethernet 110 20202021 3840 192.168.1.111 5540
```

where:

-   `110` is the assigned node id
-   `20202021` is the pin code for the bridge app
-   `3840` is the discriminator id
-   `192.168.1.111` is the IP address of the host for the bridge
-   `5540` the the port for the bridge

Alternatively, to commission with discovery which works with DNS-SD:

```bash
chip-tool pairing onnetwork 110 20202021
```

### Command

Switching on/off:

```bash
chip-tool onoff toggle 110 1 # toggle is stateless and recommended
chip-tool onoff on 110 1
chip-tool onoff off 110 1
```

where:

-   `onoff` is the matter cluster name
-   `on`/`off`/`toggle` is the command name. The `toggle` command is RECOMMENDED
    because it is stateless. The bridge does not synchronize the actual state of
    devices.
-   `110` is the node id of the bridge app assigned during the commissioning
-   `1` is the endpoint of the configured device

Level control:
```bash
chip-tool levelcontrol move-to-level 100 0 0 0 110 1
```

Color control:
```bash
chip-tool colorcontrol move-to-hue 50 0 0 0 0 110 1
chip-tool colorcontrol move-to-saturation 60 0 0 0 110 1
chip-tool colorcontrol move-to-hue-and-saturation 50 60 0 0 0 110 1
```

<!--
Color temperature:
```bash
chip-tool colorcontrol move-to-color-temperature 3000 0 0 0 110 1
```
-->

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
