#include <wiringPi.h>

int main(void)
{
    wiringPiSetup();
    int pin = 7;
    pinMode(pin, OUTPUT);
    for (;;)
    {
        digitalWrite(pin, HIGH);
        delay(500);
        digitalWrite(pin, LOW);
        delay(500);
    }
    return 0;
}
