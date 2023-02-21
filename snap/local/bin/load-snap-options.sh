#!/bin/bash -e

export GPIO=$(snapctl get gpio)

exec "$@"
