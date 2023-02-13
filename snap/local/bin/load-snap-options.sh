#!/bin/bash -e

export WIRINGPI_PIN=$(snapctl get wiringpi-pin)

exec "$@"
