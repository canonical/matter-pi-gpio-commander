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

// Environment variables
#define GPIO "GPIO"
#define GPIOCHIP "GPIOCHIP"

#define GPIO_CONSUMER "matter-commander"

static struct gpiod_line_request *lineRequest=NULL;
int gpioLine; 

static struct gpiod_line_request *
requestOutputGPIOline(const std::string& chip_path, unsigned int offset,
                      enum gpiod_line_value value)
{
	struct gpiod_request_config *req_cfg = NULL;
	struct gpiod_line_request *request = NULL;
	struct gpiod_line_settings *settings;
	struct gpiod_line_config *line_cfg;
	struct gpiod_chip *chip;
	int ret;

	chip = gpiod_chip_open(chip_path.c_str());
	if (!chip)
		return NULL;

	settings = gpiod_line_settings_new();
	if (!settings){
        ChipLogError(AppServer, "Not possible to config new gpio line, closing \
                      gpiochip");
	    gpiod_chip_close(chip);
    }

	gpiod_line_settings_set_direction(settings,
					  GPIOD_LINE_DIRECTION_OUTPUT);
	gpiod_line_settings_set_output_value(settings, value);

	line_cfg = gpiod_line_config_new();
	if (!line_cfg) {
        ChipLogError(AppServer, "Not possible to config new line, freeing \
                          setings");
	    gpiod_line_settings_free(settings);
    }
 
	ret = gpiod_line_config_add_line_settings(line_cfg, &offset, 1,
						  settings);

	req_cfg = gpiod_request_config_new();
	if (!req_cfg || ret){
        ChipLogError(AppServer, "Failed to request new config. Freeing config");
	    gpiod_line_config_free(line_cfg);
    }

	gpiod_request_config_set_consumer(req_cfg, GPIO_CONSUMER);

	request = gpiod_chip_request_lines(chip, req_cfg, line_cfg);

	gpiod_request_config_free(req_cfg);

	return request;
}


CHIP_ERROR LightingManager::Init()
{
    // ************** Configuration for GPIO LINE **************
    char *envGPIOLINE = std::getenv(GPIO);
    if (envGPIOLINE == NULL || strlen(envGPIOLINE) == 0)
    {
        ChipLogError(AppServer, "Unset or empty environment variable: %s", GPIO);
        return CHIP_ERROR_INVALID_ARGUMENT;
    }

    char *endPtr;

    gpioLine = (int)std::strtol(envGPIOLINE, &endPtr, 10); // Convert string to long, base 10

    if (*endPtr != '\0' || endPtr == envGPIOLINE)
    {
        ChipLogError(AppServer, "Failed to convert GPIO line to integer: %s", envGPIOLINE);
        // Handle the error, for example by returning an error code.
        return CHIP_ERROR_INVALID_ARGUMENT;
    }

    ChipLogProgress(AppServer, "Using GPIO line %d", gpioLine);
    // ************** Configuration for GPIO LINE **************

    // ************** Configuration for GPIO CHIP **************
    char *envGPIOCHIP = std::getenv(GPIOCHIP);
    if (envGPIOCHIP == NULL || strlen(envGPIOCHIP) == 0)
    {
        ChipLogError(AppServer, "Unset or empty environment variable: %s", GPIOCHIP);
        return CHIP_ERROR_INVALID_ARGUMENT;
    }
    ChipLogProgress(AppServer, "Using GPIOCHIP %s", envGPIOCHIP);

    std::string gpioDevice = (std::string)"/dev/gpiochip" + envGPIOCHIP;

    // ************** Configuration for GPIO CHIP **************

    enum gpiod_line_value value = GPIOD_LINE_VALUE_INACTIVE; 
    lineRequest = requestOutputGPIOline(gpioDevice, gpioLine, value);
    if (!lineRequest){
        ChipLogError(AppServer, "Failed to request gpio line:\n%s", 
                          std::strerror(errno));
        return CHIP_ERROR_INTERNAL;
    }

    // initialize both the stored and actual states to off
    mState = kState_Off;
    Set(0);

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
    enum gpiod_line_value value = aOn ? 
                    GPIOD_LINE_VALUE_ACTIVE : GPIOD_LINE_VALUE_INACTIVE; 
    int ret = gpiod_line_request_set_value(lineRequest, gpioLine, value);
    if(ret < 0){
       ChipLogError(AppServer, "Failed to set line to value: %d", value);
    }
}


