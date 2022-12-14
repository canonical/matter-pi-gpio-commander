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

from threading import Event
import os

from PyP100 import PyL530

dev = None
dev_state = None
switchedOn = None


def switch_on():
    global dev
    global switchedOn

    print("[tapo] {}: switch on".format(dev.ipAddress))
    dev.turnOn()
    switchedOn = True


def switch_off():
    global dev
    global switchedOn

    print("[tapo] {}: switch off".format(dev.ipAddress))
    dev.turnOff()
    switchedOn = False


def set_level(level: int):
    global dev
    global switchedOn

    # The level setting is stored and resubmitted with every on/off command.
    # Skip when off or unknown, because setting level turns on the Tapo light.
    if switchedOn or switchedOn is None:
        print("[tapo] {}: set brightness level".format(dev.ipAddress))
        dev.setBrightness(level)


def set_hue(hue: int):
    global dev
    global dev_state

    dev_state['hue'] = hue
    saturation = dev_state['saturation']

    print("[tapo] {}: set color {}, {}".format(dev.ipAddress, hue, saturation))
    dev.setColor(hue, saturation)


def set_saturation(saturation: int):
    global dev
    global dev_state

    dev_state['saturation'] = saturation
    hue = dev_state['hue']

    print("[tapo] {}: set color {}, {}".format(dev.ipAddress, hue, saturation))
    dev.setColor(hue, saturation)


def set_color_temperature(kelvin: int):
    global dev

    print("[tapo] {}: set color temperature".format(dev.ipAddress))
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
                global dev_state
                # hue
                if attributeId == 0:
                    print("[callback] color hue={}".format(value[0]))
                    set_hue(value[0])
                # saturation
                elif attributeId == 1:
                    print("[callback] color saturation={}".format(value[0]))
                    set_saturation(value[0])
                # temperature
                elif attributeId == 7:
                    print("[callback] color temperature={}".format(value[0]))
                    set_color_temperature(value[0])

        else:
            print("[callback] Error: unhandled cluster {} or attribute {}".format(
                clusterId, attributeId))
            pass
    else:
        print("[callback] Error: unhandled endpoint {} ".format(endpoint))


if __name__ == "__main__":
    print("Starting Tapo Bridge Lighting App")

    ip = os.environ['IP']
    user = os.environ['USER']
    password = os.environ['PASSWORD']

    dev = PyL530.L530(ip, user, password)
    print(user, password)

    print("[tapo] {}: handshake".format(ip))
    dev.handshake()
    print("[tapo] {}: login".format(ip))
    dev.login()
    print("[tapo] {}: ready ✅".format(ip))

    info = dev.getDeviceInfo()
    dev_state = info['result']['default_states']['state']
    print("[tapo] Device state:", dev_state)

    chipHandler = GetLibraryHandle(attributeChangeCallback)

    print('🚀 Ready...')
    Event().wait()
