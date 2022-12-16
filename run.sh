#!/bin/bash

set -eu

source $SNAP/connectedhomeip/python_env/bin/activate_snap

echo "venv: $VIRTUAL_ENV"

export GPIO=$(snapctl get gpio)

python3 $SNAP/bin/lighting.py
