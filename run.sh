#!/bin/bash

set -eu

source $SNAP/connectedhomeip/python_env/bin/activate_snap

echo "venv: $VIRTUAL_ENV"

export IP=$(snapctl get ip)
export USER=$(snapctl get user)
export PASSWORD=$(snapctl get password)

python3 $SNAP/bin/lighting.py
