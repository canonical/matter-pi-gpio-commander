# GPIO

## Run the blink app
```bash
sudo snap run matter-pi-gpio-commander.test-blink
```

If you get the following error, it means that the GPIO access is not allowed. Refer to [GPIO Access](#gpio-access) for details.
```
sudo snap run chip-lighting-app.test-blink
wiringPiSetup: Unable to open /dev/mem or /dev/gpiomem: Permission denied.
  Aborting your program because if it can not access the GPIO
  hardware then it most certianly won't work
  Try running with sudo?
```

## GPIO Access

This snap is strictly confined which means that the access to interfaces are subject to various security measures.

On a Linux distribution without snap confinement for GPIO (e.g. Raspberry Pi OS 11), the snap may be able to access the GPIO directly, without any snap interface and manual connections.

On Linux distributions with snap confinement for GPIO such as Ubuntu Core, the GPIO access is possible via the [gpio interface](https://snapcraft.io/docs/gpio-interface), provided by a gadget snap. 
The official [Raspberry Pi Ubuntu Core](https://ubuntu.com/download/raspberry-pi-core) image includes that gadget.

It is NOT possible to use this snap on Linux distributions that have the GPIO confinement but not the interface (e.g. Ubuntu and its flavours), unless for development purposes. In development environments, the snap may be installed in dev mode (using `--devmode` flag) which allows direct GPIO access but disables security confinement and automatic upgrades. For more details about this limitation refer [here](https://forum.snapcraft.io/t/confined-access-to-gpio-on-classic-ubuntu/29235).
