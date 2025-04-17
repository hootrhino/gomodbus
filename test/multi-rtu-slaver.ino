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
void sendReadCoilsResponse(uint8_t slaveId, uint16_t coilCount)
{
    uint8_t byteCount = (coilCount + 7) / 8;
    uint8_t responseBuffer[3 + byteCount];

    responseBuffer[0] = slaveId;
    responseBuffer[1] = 0x01;
    responseBuffer[2] = byteCount;

    for (uint8_t i = 0; i < byteCount; i++)
    {
        responseBuffer[3 + i] = 0xFF;
    }

    uint16_t crc = crc16(responseBuffer, 3 + byteCount);
    uint8_t crcLow = crc & 0xFF;
    uint8_t crcHigh = (crc >> 8) & 0xFF;
    for (uint8_t i = 0; i < (3 + byteCount); i++)
    {
        Serial.write(responseBuffer[i]);
    }

    Serial.write(crcLow);
    Serial.write(crcHigh);
}
void sendDynamicBytes(byte slaveId, unsigned short quantity)
{
    byte byteCount = quantity * 2;
    byte responseBuffer[3 + byteCount];

    responseBuffer[0] = slaveId;
    responseBuffer[1] = READ_HOLDING_REGISTERS;
    responseBuffer[2] = byteCount;

    for (int i = 0; i < byteCount; i++)
    {
        responseBuffer[3 + i] = 0xFF;
    }

    unsigned int crc = crc16(responseBuffer, 3 + byteCount);
    byte crcLow = crc & 0xFF;
    byte crcHigh = (crc >> 8) & 0xFF;

    for (int i = 0; i < (3 + byteCount); i++)
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
    switch (functionCode)
    {
    case 1: // Read Coils
        sendReadCoilsResponse(slaveId, quantity);
        break;
    case 2: // Read Discrete Inputs
        sendReadCoilsResponse(slaveId, quantity);
        break;
    case 3: // Read Holding Registers
    case 4: // Read Input Registers
        sendDynamicBytes(slaveId, quantity);
        break;
    default:
        break;
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
