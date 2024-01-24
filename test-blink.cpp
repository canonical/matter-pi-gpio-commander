// Build:
// g++ -Wall test-blink.cpp -lgpiod -o test-blink

#include <gpiod.h>
#include <cstdlib>
#include <iostream>
#include <cstring>
#include <unistd.h>

#define GPIO "GPIO"
#define CONSUMER "test-blink"

void setGpioLineValue(struct gpiod_line *line, int value);

int main(void)
{
    struct gpiod_line *line;

    char *envGPIO = std::getenv(GPIO);
    if (envGPIO == NULL || strlen(envGPIO) == 0)
    {
        std::cout << "Environment variable not set or empty: " << GPIO << std::endl;
        return 1;
    }

    int gpio;
    try
    {
        gpio = std::stoi(envGPIO);
        std::cout << "GPIO: " << gpio << std::endl;
    }
    catch (std::exception &ex)
    {
        std::cerr << "Non-integer value for GPIO: " << ex.what() << std::endl;
        return 1;
    }

    // Setup GPIO with libgpiod
    struct gpiod_chip *chip;

    chip = gpiod_chip_open("/dev/gpiochip0");
    if (!chip)
    {
        std::cerr << "Failed to open gpiochip: /dev/gpiochip0" << std::endl;
        return 1;
    }

    line = gpiod_chip_get_line(chip, gpio);
    if (!line)
    {
        std::cerr << "Failed to get line! Output code: " << envGPIO << std::endl;
        return 1;
    }

    int ret = gpiod_line_request_output(line, CONSUMER, 0);
    if (ret < 0)
    {
        std::cerr << "Request line as output failed! Output code: " << ret << std::endl;
        return 1;
    }

    for (;;)
    {
        setGpioLineValue(line, 1);
        std::cout << "On" << std::endl;
        usleep(5e5);

        setGpioLineValue(line, 0);
        std::cout << "Off" << std::endl;
        usleep(5e5);
    }

    return 0;
}

void setGpioLineValue(struct gpiod_line *line, int value)
{
   int ret = gpiod_line_set_value(line, value);
   if(ret < 0)
   {
        std::cerr << "Failed to set line to " << value << "! Output code: " << ret << std::endl;
   }
}
