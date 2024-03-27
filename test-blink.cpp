// Build:
// Dynamically
// g++ -Wall test-blink.cpp -lgpiod -o test-blink
// Statically (Used in this snap)
// g++ -Wall -I<path to gpiod.h dir> -L<path to libgpiod.a dir> test-blink.cpp -lgpiod -static -o test-bin

#include <gpiod.h>
#include <cstdlib>
#include <iostream>
#include <cstring>
#include <unistd.h>

// Environment variables
#define GPIO "GPIO"
#define GPIOCHIP "GPIOCHIP"

#define GPIO_CONSUMER "matter-commander-test-blink"

static struct gpiod_line_request *
requestOutputGPIOline(std::string *chip_path, unsigned int offset,
		    enum gpiod_line_value value)
{
	struct gpiod_request_config *req_cfg = NULL;
	struct gpiod_line_request *request = NULL;
	struct gpiod_line_settings *settings;
	struct gpiod_line_config *line_cfg;
	struct gpiod_chip *chip;
	int ret;

	chip = gpiod_chip_open(chip_path->c_str());
	if (!chip)
		return NULL;

	settings = gpiod_line_settings_new();
	if (!settings){
        std::cerr << "Error: not possible to config new line, closing chip" 
                      << std::endl;
	    gpiod_chip_close(chip);
    }

	gpiod_line_settings_set_direction(settings,
					  GPIOD_LINE_DIRECTION_OUTPUT);
	gpiod_line_settings_set_output_value(settings, value);

	line_cfg = gpiod_line_config_new();
	if (!line_cfg) {
        std::cerr <<  "Error: not possible to config new line, freeing setings" 
                          << std::endl;
	    gpiod_line_settings_free(settings);
    }
 
	ret = gpiod_line_config_add_line_settings(line_cfg, &offset, 1,
						  settings);

	req_cfg = gpiod_request_config_new();
	if (!req_cfg || ret){
        std::cerr << "Error: Not possible to request new config. Freeing line \
                          config" << std::endl;
	    gpiod_line_config_free(line_cfg);
    }

	gpiod_request_config_set_consumer(req_cfg, GPIO_CONSUMER);

	request = gpiod_chip_request_lines(chip, req_cfg, line_cfg);

	gpiod_request_config_free(req_cfg);

	return request;
}

static enum gpiod_line_value toggle_line_value(enum gpiod_line_value value)
{
	return (value == GPIOD_LINE_VALUE_ACTIVE) ? GPIOD_LINE_VALUE_INACTIVE :
						    GPIOD_LINE_VALUE_ACTIVE;
}

int main(void)
{

    // Read environment variables
    char *envGPIO = std::getenv(GPIO);
    if (envGPIO == NULL || strlen(envGPIO) == 0)
    {
        std::cout << "Unset or empty environment variable: " << GPIO 
            << std::endl;
        return 1;
    }
    std::cout << "GPIO: " << envGPIO << std::endl;

    char *envGPIOCHIP = std::getenv(GPIOCHIP);
    if (envGPIOCHIP == NULL || strlen(envGPIOCHIP) == 0)
    {
        std::cout << "Unset or empty environment variable: " << GPIOCHIP 
            << std::endl;
        return 1;
    }
    std::cout << "GPIOCHIP: " << envGPIOCHIP << std::endl;

    int gpioLine;
    try
    {
        gpioLine = std::stoi(envGPIO);
    }
    catch (std::exception &ex)
    {
        std::cerr << "Non-integer value for GPIO: " << ex.what() << std::endl;
        return 1;
    }

    std::string gpioDevice = (std::string)"/dev/gpiochip" + envGPIOCHIP;
    

    struct gpiod_line_request *request;
    enum gpiod_line_value value = GPIOD_LINE_VALUE_ACTIVE; 

    request = requestOutputGPIOline(&gpioDevice, gpioLine, value);
    if (!request) {
        std::cerr << "Failed to request output line.\n"<< std::strerror(errno) 
            << std::endl;
        return 1;
    }

    std::string msg;
    for (;;)
    {

        value = toggle_line_value(value);
        std::cout << "Setting GPIO " + std::to_string(gpioLine) + " to " + 
            ((value == GPIOD_LINE_VALUE_ACTIVE)? "On" : "Off") << std::endl;
        gpiod_line_request_set_value(request, gpioLine, value);
        usleep(5e5); 
    }

    gpiod_line_request_release(request);

    return 0;
}