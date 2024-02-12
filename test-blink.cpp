// Build:
// g++ -Wall test-blink.cpp -lgpiod -o test-blink

#include <gpiod.h>
#include <cstdlib>
#include <iostream>
#include <cstring>
#include <unistd.h>

// Environment variables
#define GPIO "GPIO"
#define GPIOCHIP "GPIOCHIP"

#define GPIO_CONSUMER "matter-commander-test-blink"

void setGpioLineValue(struct gpiod_line *line, int value);

int main(void)
{
    struct gpiod_line *line;

    // Read environment variables
    char *envGPIO = std::getenv(GPIO);
    if (envGPIO == NULL || strlen(envGPIO) == 0)
    {
        std::cout << "Unset or empty environment variable: " << GPIO << std::endl;
        return 1;
    }
    std::cout << "GPIO: " << envGPIO << std::endl;

    char *envGPIOCHIP = std::getenv(GPIOCHIP);
    if (envGPIOCHIP == NULL || strlen(envGPIOCHIP) == 0)
    {
        std::cout << "Unset or empty environment variable: " << GPIOCHIP << std::endl;
        return 1;
    }
    std::cout << "GPIOCHIP: " << envGPIOCHIP << std::endl;

    // Convert
    int gpio;
    try
    {
        gpio = std::stoi(envGPIO);
    }
    catch (std::exception &ex)
    {
        std::cerr << "Non-integer value for GPIO: " << ex.what() << std::endl;
        return 1;
    }

    std::string gpioDevice = (std::string)"/dev/gpiochip" + envGPIOCHIP;
    
    // Setup GPIO with libgpiod
    struct gpiod_chip *chip;

    chip = gpiod_chip_open(gpioDevice.c_str());
    if (!chip)
    {
        std::cerr << "Failed to open gpio chip: " << gpioDevice << std::endl;
        return 1;
    }

    line = gpiod_chip_get_line(chip, gpio);
    if (!line)
    {
        std::cerr << "Failed to get gpio line: " << gpio << std::endl;
        return 1;
    }

    int ret = gpiod_line_request_output(line, GPIO_CONSUMER, 0);
    if (ret < 0)
    {
        std::cerr << "Failed to set gpio line as output." << std::endl;
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
        std::cerr << "Failed to set gpio line to " << value << std::endl;
   }
}
