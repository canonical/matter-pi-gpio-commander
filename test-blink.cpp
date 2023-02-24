// Dependency:
// https://github.com/WiringPi/WiringPi
//
// Build:
// g++ -Wall test-blink.cpp -lwiringPi -o test-blink

#include <wiringPi.h>
#include <cstdlib>
#include <iostream>
#include <cstring>

#define GPIO "GPIO"

int main(void)
{
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

    wiringPiSetupGpio();
    pinMode(gpio, OUTPUT);

    for (;;)
    {
        digitalWrite(gpio, HIGH);
        std::cout << "On" << std::endl;
        delay(500);

        digitalWrite(gpio, LOW);
        std::cout << "Off" << std::endl;
        delay(500);
    }

    return 0;
}
