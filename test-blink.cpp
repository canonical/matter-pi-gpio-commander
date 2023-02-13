#include <wiringPi.h>
#include <cstdlib>
#include <iostream>

#define WiringPiPin "WIRINGPI_PIN"

int main(void)
{
    char *envWiringPiPin = std::getenv(WiringPiPin);
    int wiringPiPin = std::stoi(envWiringPiPin);

    std::cout << "WiringPi Pin: " << wiringPiPin << std::endl;

    wiringPiSetup();
    pinMode(wiringPiPin, OUTPUT);

    for (;;)
    {
        digitalWrite(wiringPiPin, HIGH);
        delay(500);
        digitalWrite(wiringPiPin, LOW);
        delay(500);
    }
    return 0;
}
