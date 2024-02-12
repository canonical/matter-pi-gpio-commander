# Run Tests

This section provides documentation on running tests and various scenarios.

## Environment Variables

To run the tests, you must set the following environment variables:

- `SERVICE_CHANNEL`: The channel from which the snap will be downloaded if not using a local snap file. The default option is `latest/edge`.
- `LOCAL_SERVICE_SNAP`: Path to the local service snap to be tested instead of downloading from a channel.
- `SKIP_TEARDOWN_REMOVAL`: Skip the removal of snaps during teardown (useful when running on CI machines). The default option is `false`.
- `USE_GPIO_MOCK`: This is used to determine whether you wish to use gpio-mock to test the application instead of a physical gpiochip. Possible values are `true` or `false`. Important note: the usage of gpio-mock will only work if the snap is being installed from a local file; in other words, `LOCAL_SERVICE_SNAP` must be defined.
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
