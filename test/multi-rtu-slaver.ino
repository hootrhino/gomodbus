#include <Arduino.h>
const int BUFFER_SIZE = 64;
const byte READ_HOLDING_REGISTERS = 0x03;

byte receiveBuffer[BUFFER_SIZE];
int receiveIndex = 0;

void setup()
{
    Serial.begin(9600);
}

unsigned inline int calculateCRC16(byte *data, int length)
{
    unsigned int crc = 0xFFFF;
    for (int i = 0; i < length; i++)
    {
        crc ^= (unsigned int)data[i];
        for (int j = 0; j < 8; j++)
        {
            if (crc & 0x0001)
            {
                crc >>= 1;
                crc ^= 0xA001;
            }
            else
            {
                crc >>= 1;
            }
        }
    }
    return crc;
}

void processModbusRequest()
{
    byte slaveId = receiveBuffer[0];
    byte functionCode = receiveBuffer[1];
    if (functionCode == READ_HOLDING_REGISTERS)
    {
        if (slaveId >= 1 && slaveId <= 255)
        {
            sendModbusResponse(slaveId);
        }
    }
}

void sendModbusResponse(byte slaveId)
{
    byte responseBuffer[5];
    responseBuffer[0] = slaveId;
    responseBuffer[1] = READ_HOLDING_REGISTERS;
    responseBuffer[2] = 0x02;
    responseBuffer[3] = 0xAB;
    responseBuffer[4] = 0xFF;

    unsigned int crc = calculateCRC16(responseBuffer, 5);
    byte crcLow = crc & 0xFF;
    byte crcHigh = (crc >> 8) & 0xFF;

    for (int i = 0; i < 5; i++)
    {
        Serial.write(responseBuffer[i]);
    }
    Serial.write(crcLow);
    Serial.write(crcHigh);
}

void loop()
{
    while (Serial.available() > 0)
    {
        byte incomingByte = Serial.read();
        receiveBuffer[receiveIndex++] = incomingByte;
        if (receiveIndex >= 8)
        {
            processModbusRequest();
            receiveIndex = 0;
        }
    }
    delay(30);
}
