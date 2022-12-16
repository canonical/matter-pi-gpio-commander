# Matter Pi GPIO Commander
This app is a Matter lighting device which can be used to control LED using Raspberry Pi's GPIO pin.

The lighting device communicates with others over WiFi/Ethernet.
## Snap
### Build and install
```bash
snapcraft -v
snap install --dangerous ./matter-lighting-gpio_0.1_arm64.snap
```
### Configure
```bash
snap set matter-lighting-gpio gpio=17
```

### Connect interfaces
```bash
snap connect matter-lighting-gpio:avahi-control
```

The [avahi-control](https://snapcraft.io/docs/avahi-control-interface) is necessary to allow discovery of the application via DNS-SD.
To make this work, the host also needs to have a running avahi-daemon which can be installed with `sudo apt install avahi-daemon` or `snap install avahi`.

### Run
```bash
sudo snap start matter-lighting-gpio
sudo snap logs -f matter-lighting-gpio
```

## Native

Assuming you have setup the Connected Home IP project for Python projects (see [Development](#development)) at `../connectedhomeip`:

### Activate the Python env
```bash
source ../connectedhomeip/out/python_env/bin/activate
```

### Run
```bash
GPIO=17 python lighting.py
```

## Control with Chip Tool

### Commissioning

```bash
chip-tool pairing ethernet 110 20202021 3840 192.168.1.111 5540
```

where:

-   `110` is the assigned node id
-   `20202021` is the pin code for the lighting app
-   `3840` is the discriminator id
-   `192.168.1.111` is the IP address of the host for the lighting app
-   `5540` the the port for the lighting app

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
