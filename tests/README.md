# Run Tests

This section provides documentation on running tests and various scenarios.

## Environment Variables

To run the tests, you must set the following environment variables:

- `SERVICE_CHANNEL`: The channel from which the snap will be downloaded if not using a local snap file. The default option is `latest/edge`.
- `LOCAL_SERVICE_SNAP`: Path to the local service snap to be tested instead of downloading from a channel.
- `SKIP_TEARDOWN_REMOVAL`: Skip the removal of snaps during teardown (useful when running on CI machines). The default option is `false`.
- `MOCK_GPIO`: This is used to determine whether you wish to use gpio-mock to test the application instead of a physical gpiochip. Possible values are `true` or `false`. Important note: the usage of gpio-mock will only work if the snap is being installed from a local file; in other words, `LOCAL_SERVICE_SNAP` must be defined.
- `GPIO_CHIP`: The GPIO chip number; accepted values are `0` (for legacy Raspberry Pis) or `4` for the Raspberry Pi 5. This doesn't need to be specified if using `USE_GPIO_MOCK`.
- `GPIO_LINE`: This is the line offset to be used to test the selected gpiochip. The number of available lines can be checked with the `gpiodetect` and `gpioinfo` commands from the Debian package `gpiod`. This doesn't need to be specified if using `USE_GPIO_MOCK`.

**Note:** The `USE_GPIO_MOCK` takes precedence over the specific `GPIO_CHIP` and `GPIO_LINE` settings if the former is set to "1"; thus, the other two are ignored.

## Running tests

```bash
go test -v -failfast -count 1
```

where:
- `-v` is to enable verbose output
- `-failfast` makes the test stop after first failure
- `-count 1` is to avoid Go test caching for example when testing a rebuilt snap

# Thread Tests

These tests are meant to be run manually; therefore, they are in this separate 
directory.

## Environment Variables

Some environment variables are needed to run the tests. The main reason is that
the tests actually run on two devices, each with two Radio Co-Processors (RCPs)
(such as the nRF52480 dongle) attached to them, and they communicate using an
SSH connection. The machine used to run the test will be referred to as
"first machine" or **machine (A)**, and the machine accessed by SSH will be
referred to as "second machine" or **machine (B)**.

You can refer to [this guide][openthread-border-router-snap-guide-url] to learn
how to build and flash an RCP firmware.

* `REMOTE_USER` - The user to be logged in on the second machine **(B)**.
* `REMOTE_PASSWORD` - The password for the second machine **(B)** user.
* `REMOTE_HOST` - The IP address of the second machine **(B)**.
* `REMOTE_INFRA_IF` - The network interface name for the second machine **(B)**.
* `LOCAL_INFRA_IF` - The network interface name for the first machine **(A)**.
* `REMOTE_SNAP_PATH` - The path to the snap file on the second machine **(B)**,
if doesn't specified, snap if fetched from store.
* `REMOTE_GPIO_CHIP` - The number for the GPIO chip to be used on **(B)**.
* `REMOTE_GPIO_LINE` - The number for the GPIO line to be used on **(B)**.

Example:

```bash
REMOTE_SNAP_PATH="~/matter-pi-gpio-commander_2.0.0_arm64.snap" \
REMOTE_GPIO_CHIP="0" \
REMOTE_GPIO_LINE="16" \
LOCAL_INFRA_IF="eno1" \
REMOTE_INFRA_IF="eth0" \
REMOTE_USER="ubuntu" \
REMOTE_PASSWORD="abcdef" \
REMOTE_HOST="192.168.178.95" \
go test -v -failfast -count 0 ./thread_tests
```

[openthread-border-router-snap-guide-url]: https://github.com/canonical/openthread-border-router-snap/wiki/Setup-OpenThread-Border-Router-with-nRF52840-Dongle#build-and-flash-rcp-firmware-on-nrf52480-dongle