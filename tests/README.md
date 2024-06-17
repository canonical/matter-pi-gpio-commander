# Tests

Refer below for instructions on running the snap tests.
There are two different testing suites with different hardware requirements and inputs.

## Run WiFi tests

The WiFi tests run the Matter controller along with this snap on the same machine. 
The machine should be a compatible Raspberry Pi, unless the GPIO is mocked.

To run the tests, you must set the following environment variables:

- `SERVICE_CHANNEL`: The channel from which the snap will be downloaded. The default value is `latest/edge`. This is ignored when using a locally built snap.
- `LOCAL_SERVICE_SNAP`: Path to the local service snap to be tested instead of downloading from a channel.
- `SKIP_TEARDOWN_REMOVAL`: Skip the removal of snaps during teardown (useful when running on CI machines). The default value is `false`.
- `MOCK_GPIO`: Use gpio-mock to test the application instead of a physical gpiochip. The default value is `false`. The GPIO mocking logic works by modifying a local snap; the path to which must be set in `LOCAL_SERVICE_SNAP`.
- `GPIO_CHIP`: The GPIO chip number; accepted values are `0` (for legacy Raspberry Pis) or `4` for the Raspberry Pi 5. This is ignored when mocking GPIO.
- `GPIO_LINE`: This is the line offset to be used to test the selected gpiochip. The number of available lines can be checked with the `gpiodetect` and `gpioinfo` commands from the Debian package `gpiod`. This is ignored when mocking GPIO.

Example, for running tests on a Raspberry Pi 4:

```bash
GPIO_CHIP=0 \
GPIO_LINE=16 \
go test -v -failfast -count 1
```

where:
- `-v` is to enable verbose output
- `-failfast` makes the test stop after first failure
- `-count 1` is to avoid Go test caching when repeating the unchanged tests, such when re-testing a rebuilt snap.

# Run Thread tests

These tests have additional hardware dependencies:
- A nRF52480 dongle with OT RCP firmware - connected to the local machine
- A compatible Raspberry Pi - used as the remote device
- A second nRF52480 dongle with OT RCP firmware - connected to the Raspberry Pi

You can refer to [this guide][openthread-border-router-snap-guide-url] to learn how to build and flash an RCP firmware.

The tests will configure the remote device over SSH: an open SSH server with password-login on the Raspberry Pi is required.

Additional environment variables needed for these tests:
* For the remote machine:
  * `REMOTE_HOST`: IP address or hostname
  * `REMOTE_USER`: username for SSH login
  * `REMOTE_PASSWORD`: password for the user
  * `REMOTE_INFRA_IF`: network interface name used for the main networking interface on the remote machine (usually the Wifi or Ethernet interface)
  * `REMOTE_GPIO_CHIP`: GPIO chip number
  * `REMOTE_GPIO_LINE`: GPIO line number
  * `REMOTE_SNAP_PATH`: path to the snap file. If doesn't specified, snap if fetched from store.
* For the local machine:
  * `LOCAL_INFRA_IF`: network interface name used for the main networking interface on the local machine (usually the Wifi or Ethernet interface)

Example, for testing a locally built snap, available on the remote machine at `~/matter-pi-gpio-commander_2.0.0_arm64.snap`:

```bash
REMOTE_HOST="192.168.178.95" \
REMOTE_USER="ubuntu" \
REMOTE_PASSWORD="abcdef" \
REMOTE_INFRA_IF="eth0" \
REMOTE_GPIO_CHIP="0" \
REMOTE_GPIO_LINE="16" \
REMOTE_SNAP_PATH="~/matter-pi-gpio-commander_2.0.0_arm64.snap" \
LOCAL_INFRA_IF="eno1" \
go test -v -failfast -count 1 ./thread_tests
```

[openthread-border-router-snap-guide-url]: https://github.com/canonical/openthread-border-router-snap/wiki/Setup-OpenThread-Border-Router-with-nRF52840-Dongle
