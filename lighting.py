#
#    Copyright (c) 2021 Project CHIP Authors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#

# To run, execute the following inside the Python Virtual Env:
# pip install PyP100
# IP="" USER="" PASS="" python lighting.py

from chip.server import (
    GetLibraryHandle,
    NativeLibraryHandleMethodArguments,
    PostAttributeChangeCallback,
)

from chip.exceptions import ChipStackError


import os
import asyncio

from PyP100 import PyL530

dev = None
color = {}
switchedOn = None


def switch_on():
    global dev
    global switchedOn

    print("[tapo] switch on")
    dev.turnOn()
    switchedOn = True


def switch_off():
    global dev
    global switchedOn

    print("[tapo] switch off")
    dev.turnOff()
    switchedOn = False


def set_level(level: int):
    global dev
    global switchedOn

    # The level setting is stored and resubmitted with every on/off command.
    # Skip when off or unknown, because setting level turns on the Tapo light.
    if switchedOn or switchedOn is None:
        print("[tapo] set brightness level")
        dev.setBrightness(level)


def set_color(level: dict):
    global dev

    print("[tapo] set color")
    dev.setColor(level['hue'], level['saturation'])


def set_color_temperature(kelvin: int):
    global dev

    print("[tapo] set color temperature")
    dev.setColorTemp(kelvin)


@PostAttributeChangeCallback
def attributeChangeCallback(
    endpoint: int,
    clusterId: int,
    attributeId: int,
    xx_type: int,
    size: int,
    value: bytes,
):
    if endpoint == 1:
        print("[callback] cluster={} attr={} value={}".format(
            clusterId, attributeId, list(value)))
        # switch
        if clusterId == 6 and attributeId == 0:
            if value and value[0] == 1:
                print("[callback] light on")
                switch_on()
            else:

                print("[callback] light off")
                switch_off()
        # level (brightness)
        elif clusterId == 8 and attributeId == 0:
            if value:
                print("[callback] level {}".format(value[0]))
                set_level(value[0])
        # color
        elif clusterId == 768:
            if value:
                global color
                if attributeId == 0:
                    print("[callback] color hue={}".format(value[0]))
                    color['hue'] = value[0]
                elif attributeId == 1:
                    print("[callback] color saturation={}".format(value[0]))
                    color['saturation'] = value[0]
                elif attributeId == 7:
                    print("[callback] color temperature={}".format(value[0]))
                    set_color_temperature(value[0])

                # we need both hue and saturation to set a new color
                if (attributeId == 0 or attributeId == 1) and 'hue' in color and 'saturation' in color:
                    print("[callback] color={}".format(color))
                    set_color(color)
        else:
            print("[callback] Error: unhandled cluster {} or attribute {}".format(
                clusterId, attributeId))
            pass
    else:
        print("[callback] Error: unhandled endpoint {} ".format(endpoint))


class Lighting:
    def __init__(self):
        self.chipLib = GetLibraryHandle(attributeChangeCallback)


if __name__ == "__main__":
    l = Lighting()

    print("Starting Tapo Bridge Lighting App")

    ip = os.environ['IP']
    user = os.environ['USER']
    password = os.environ['PASS']

    dev = PyL530.L530(ip, user, password)

    print("[tapo] handshake")
    dev.handshake()
    print("[tapo] login")
    dev.login()

    print("[tapo] ready")

    loop = asyncio.get_event_loop()
    try:
        loop.run_forever()
    except KeyboardInterrupt:
        print("Process interrupted")
    finally:
        loop.close()
        print("Shutting down")
