#include <Arduino.h>
const int BUFFER_SIZE = 64;
const byte READ_HOLDING_REGISTERS = 0x03;

byte receiveBuffer[BUFFER_SIZE];
int receiveIndex = 0;

void setup()
{
    Serial.begin(9600);
}

unsigned inline int crc16(byte *data, int length)
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

void send2Bytes(byte slaveId)
{
    byte responseBuffer[5];
    responseBuffer[0] = slaveId;
    responseBuffer[1] = READ_HOLDING_REGISTERS;
    responseBuffer[2] = 0x02;
    responseBuffer[3] = 0xAB;
    responseBuffer[4] = 0xFF;

    unsigned int crc = crc16(responseBuffer, 5);
    byte crcLow = crc & 0xFF;
    byte crcHigh = (crc >> 8) & 0xFF;

    for (int i = 0; i < 5; i++)
    {
        Serial.write(responseBuffer[i]);
    }
    Serial.write(crcLow);
    Serial.write(crcHigh);
}
void send4Bytes(byte slaveId)
{
    byte responseBuffer[7]; // Send 4 bytes of data
    responseBuffer[0] = slaveId;
    responseBuffer[1] = READ_HOLDING_REGISTERS;
    // 3.14159= 40 49 0F DA
    responseBuffer[2] = 0x04;
    responseBuffer[3] = 0x40;
    responseBuffer[4] = 0x49;
    responseBuffer[5] = 0x0F;
    responseBuffer[6] = 0xDA;
    unsigned int crc = crc16(responseBuffer, 7);
    byte crcLow = crc & 0xFF;
    byte crcHigh = (crc >> 8) & 0xFF;
    for (int i = 0; i < 7; i++)
    {
        Serial.write(responseBuffer[i]);
    }
    Serial.write(crcLow);
    Serial.write(crcHigh);
}

void processModbusRequest()
{
    byte slaveId = receiveBuffer[0];
    byte functionCode = receiveBuffer[1];
    byte startAddressHigh = receiveBuffer[2];
    byte startAddressLow = receiveBuffer[3];
    byte quantityHigh = receiveBuffer[4];
    byte quantityLow = receiveBuffer[5];
    byte crcLow = receiveBuffer[6];
    byte crcHigh = receiveBuffer[7];
    unsigned short quantity = (quantityHigh << 8) | quantityLow;
    if (quantity == 1)
    {
        send2Bytes(slaveId);
    }
    else if (quantity == 2)
    {
        send4Bytes(slaveId);
    }
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
