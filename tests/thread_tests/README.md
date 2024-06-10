# Thread Tests

This tests are meant to be run manually, for this reason, they are on this
separated directory.

## Environment Variables

Some enviroment variables are needed to run the test, the main reason is that, 
the tests run actually in two devices, and those devices need to have two 
Radio Co-Processors (RCPs) (like the nRF52480 dongle for example) atached to them, 
and comunicate between then using a SSH connection the machine that is used to run
the test will be refered as "first machine" or machine(A) and the machine that will
be accessed by SSH will be refered as "second machine" or machine(B).

You can refer to [this guide](https://github.com/canonical/openthread-border-router-snap/wiki/Setup-OpenThread-Border-Router-with-nRF52840-Dongle#build-and-flash-rcp-firmware-on-nrf52480-dongle),
to know how to build and flash a RCP firmware.

* `REMOTE_USER` - The user to be logged on the second machine**(B)**.  
* `REMOTE_PASSWORD` - The password to sedond's machine**(B)** user.
* `REMOTE_HOST`- The ip address to the second machine**(B)**.
* `REMOTE_INFRA_IF` - The network interface name for the second machine**(B)**.
* `LOCAL_INFRA_IF` - The netowrk interface name for the first machine**(A)**.
* `REMOTE_SNAP_PATH` - The path to the snap file on the second machine**(B)**.
* `REMOTE_GPIO_CHIP` - **The number** for the **GPIO chip** to be used on (B), defaults to `0` which is `/dev/gpiochip0` if not defined.
* `REMOTE_GPIO_LINE` - **The number** for the **GPIO line** to be used on (B), defaults to `16` if not defined.

Example:

```bash
REMOTE_SNAP_PATH="~/matter-pi-gpio-commander_2.0.0_arm64.snap" LOCAL_INFRA_IF="eno1" REMOTE_INFRA_IF="eth0" REMOTE_USER="ubuntu" REMOTE_PASSWORD="abcdef" REMOTE_HOST="192.168.178.95" go test -v -failfast -count 1 ./thread_tests
```