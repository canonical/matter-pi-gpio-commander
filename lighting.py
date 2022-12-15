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

from chip.server import (
    GetLibraryHandle,
    PostAttributeChangeCallback,
)

import os
import sys

from gpiozero import LED

def switch_on():
    global switchedOn

    print("[LED] switch on")
    led.on()


def switch_off():
    global switchedOn

    print("[LED] switch off")
    led.off()

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
        else:
            print("[callback] Error: unhandled cluster {} or attribute {}".format(
                clusterId, attributeId))
            pass
    else:
        print("[callback] Error: unhandled endpoint {} ".format(endpoint))


class Lighting:
    def __init__(self):
        self.chipLib = GetLibraryHandle(attributeChangeCallback)

gpio=os.environ['GPIO']
led = LED(gpio)

l = Lighting()
print('🚀 Ready...')

input('Press enter to quit')
sys.exit(0)
