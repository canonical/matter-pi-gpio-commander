#!/bin/bash -e

export WIRINGPI_PIN=$(snapctl get wiringpi-pin)
export ARGS=$(snapctl get args)

exec "$@"
