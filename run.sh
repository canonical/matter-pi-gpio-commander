#!/bin/bash

set -ev

source $SNAP/connectedhomeip/python_env/bin/activate_snap

echo "venv: $VIRTUAL_ENV"

IP="192.168.1.118" USER="" PASS="" python3 $SNAP/bin/lighting.py
