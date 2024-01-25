/*
 * This file contains partial code copied from:
 * https://github.com/project-chip/connectedhomeip/blob/v1.2.0.1/examples/lighting-app/lighting-common/src/LightingManager.cpp
 */

/*
 *
 *    Copyright (c) 2020 Project CHIP Authors
 *    Copyright (c) 2019 Google LLC.
 *    All rights reserved.
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

#include "LightingManager.h"

#include <lib/support/logging/CHIPLogging.h>
#include <gpiod.h>
#include <unistd.h>
#include <cstdlib>
#include <iostream>
#include <cstring>

LightingManager LightingManager::sLight;

#define GPIO "GPIO"
#define CONSUMER "Lighting_Manager"

static int gpio;
static struct gpiod_line *gpioLine;

CHIP_ERROR LightingManager::Init()
{
    char *envGPIO = std::getenv(GPIO);
    if (envGPIO == NULL || strlen(envGPIO) == 0)
    {
        ChipLogError(AppServer, "Environment variable not set or empty: %s", GPIO);

        return CHIP_ERROR_INVALID_ARGUMENT;
    }

    ChipLogProgress(AppServer, "Using GPIO %s", envGPIO);

    gpio = std::stoi(envGPIO);

    struct gpiod_chip *chip;
    chip = gpiod_chip_open("/dev/gpiochip0");
    if (!chip)
    {
        ChipLogError(AppServer, "Failed to open gpiochip: /dev/gpiochip0");
        return CHIP_ERROR_INTERNAL;
    }

    gpioLine = gpiod_chip_get_line(chip, gpio);
    if (!gpioLine)
    {
        ChipLogError(AppServer, "Failed to get line: %s", envGPIO);
        return CHIP_ERROR_INTERNAL;
    }

    int ret = gpiod_line_request_output(gpioLine, CONSUMER, 0);
    if (ret < 0)
    {
        ChipLogError(AppServer, "Request line as output failed! Ouput code: %d", ret);
        return CHIP_ERROR_INTERNAL;
    }

    // initialize both the stored and actual states to off
    mState = kState_Off;
    SetGpioLineValue(0);

    return CHIP_NO_ERROR;
}

bool LightingManager::IsTurnedOn()
{
    return mState == kState_On;
}

void LightingManager::SetCallbacks(LightingCallback_fn aActionInitiated_CB, LightingCallback_fn aActionCompleted_CB)
{
    mActionInitiated_CB = aActionInitiated_CB;
    mActionCompleted_CB = aActionCompleted_CB;
}

bool LightingManager::InitiateAction(Action_t aAction)
{
    // TODO: this function is called InitiateAction because we want to implement some features such as ramping up here.
    bool action_initiated = false;
    State_t new_state;

    switch (aAction)
    {
    case ON_ACTION:
        ChipLogProgress(AppServer, "LightingManager::InitiateAction(ON_ACTION)");
        break;
    case OFF_ACTION:
        ChipLogProgress(AppServer, "LightingManager::InitiateAction(OFF_ACTION)");
        break;
    default:
        ChipLogProgress(AppServer, "LightingManager::InitiateAction(unknown)");
        break;
    }

    // Initiate On/Off Action only when the previous one is complete.
    if (mState == kState_Off && aAction == ON_ACTION)
    {
        action_initiated = true;
        new_state = kState_On;
    }
    else if (mState == kState_On && aAction == OFF_ACTION)
    {
        action_initiated = true;
        new_state = kState_Off;
    }

    if (action_initiated)
    {
        if (mActionInitiated_CB)
        {
            mActionInitiated_CB(aAction);
        }

        Set(new_state == kState_On);

        if (mActionCompleted_CB)
        {
            mActionCompleted_CB(aAction);
        }
    }

    return action_initiated;
}

void LightingManager::Set(bool aOn)
{
   mState = aOn ? kState_On : kState_Off;
   SetGpioLineValue(aOn ? 1 : 0);
}

void LightingManager::SetGpioLineValue(int value)
{
   int ret = gpiod_line_set_value(gpioLine, value);
   if(ret < 0)
   {
       ChipLogError(AppServer, "Failed to set line to value: %d", ret);
   }
}
