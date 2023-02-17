# GPIO

This snap is strictly confined which means that the access to interfaces are subject to various security measures.

On a Linux distribution without snap confinement for GPIO (e.g. **Raspberry Pi OS** 11), the snap may be able to access the GPIO directly, without any snap interface and manual connections.

On Linux distributions with snap confinement for GPIO such as **Ubuntu Core**, the GPIO access is possible via the [gpio interface](https://snapcraft.io/docs/gpio-interface), provided by a gadget snap. 
The official [Raspberry Pi Ubuntu Core](https://ubuntu.com/download/raspberry-pi-core) image includes that gadget.

It is NOT possible to use this snap on Linux distributions that have the GPIO confinement but not the interface (e.g. **Ubuntu** and its flavours), unless for development purposes. In development environments, the snap may be installed in dev mode (using `--devmode` flag) which allows direct GPIO access but disables security confinement and automatic upgrades. For more details about this limitation, refer [here](https://forum.snapcraft.io/t/confined-access-to-gpio-on-classic-ubuntu/29235).

The following table lists some examples:
| Distro             | GPIO Confinement | GPIO Interface | Working Installation Mode |
|--------------------|------------------|----------------|---------------------------|
| Ubuntu 22.04       | Y                | N              | Dev Mode                  |
| Ubuntu Core 22     | Y                | Y              | Default                   |
| Raspberry Pi OS 11 | N                | N              | Default                   |


Use the [Test Blink app](README.md#test-blink) to verify the installation and GPIO access.

