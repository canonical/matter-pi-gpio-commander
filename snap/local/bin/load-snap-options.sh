#!/bin/bash -e

export GPIO=$(snapctl get gpio)
export ARGS=$(snapctl get args)

exec "$@"
