#!/bin/bash -e

export GPIO=$(snapctl get gpio)
export GPIOCHIP=$(snapctl get gpiochip)
export ARGS=$(snapctl get args)

exec "$@"
