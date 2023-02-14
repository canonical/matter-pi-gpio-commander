// Dependency:
// https://github.com/WiringPi/WiringPi
//
// Build:
// g++ -Wall test-blink.cpp -lwiringPi -o test-blink

#include <wiringPi.h>
#include <cstdlib>
#include <iostream>

#define WiringPiPin "WIRINGPI_PIN"

int main(void)
{
    char *envWiringPiPin = std::getenv(WiringPiPin);
    int wiringPiPin;
    try
    {
        wiringPiPin = std::stoi(envWiringPiPin);
        std::cout << "WiringPi pin: " << wiringPiPin << std::endl;
    }
    catch (std::exception &ex)
    {
        std::cerr << "Non-integer value for WiringPi pin: " << ex.what() << std::endl;
        return 1;
    }

    wiringPiSetup();
    pinMode(wiringPiPin, OUTPUT);

    for (;;)
    {
        digitalWrite(wiringPiPin, HIGH);
        std::cout << "On" << std::endl;
        delay(500);

        digitalWrite(wiringPiPin, LOW);
        std::cout << "Off" << std::endl;
        delay(500);
    }

    return 0;
}
