/*
 * Modbus RTU Slave Simulator
 * Always returns valid response
 * Coils = ON (1), Registers = 0xABCD
 * Baudrate: 9600-8-N-1
 */

#define SLAVE_ID 0x01
#define SERIAL_BAUD 9600
#define BUF_SIZE 128

uint8_t request[BUF_SIZE];
uint8_t response[BUF_SIZE];

void setup()
{
    Serial.begin(SERIAL_BAUD);
}

void loop()
{
    if (Serial.available() >= 8)
    {
        int len = readModbusRequest();
        if (len >= 8 && verifyCRC(request, len))
        {
            buildResponse(len);
            sendResponse();
        }
    }
}

// Read Modbus request from Serial
int readModbusRequest()
{
    int i = 0;
    unsigned long start = millis();
    while ((millis() - start) < 10 && i < BUF_SIZE)
    {
        if (Serial.available())
        {
            request[i++] = Serial.read();
        }
    }
    return i;
}

// CRC16 check
bool verifyCRC(uint8_t *data, int len)
{
    uint16_t crc = modbusCRC(data, len - 2);
    return (data[len - 2] == (crc & 0xFF)) && (data[len - 1] == (crc >> 8));
}

// Build response based on function code
void buildResponse(int len)
{
    uint8_t func = request[1];
    uint16_t startAddr = (request[2] << 8) | request[3];
    uint16_t quantity = (request[4] << 8) | request[5];

    response[0] = request[0]; // Echo any Slave ID
    response[1] = func;
    if (request[0] == 0x00)
        return; // Do not respond to broadcast
    switch (func)
    {
    case 0x01: // Read Coils
    case 0x02: // Read Discrete Inputs
    {
        int byteCount = (quantity + 7) / 8;
        response[2] = byteCount;
        for (int i = 0; i < byteCount; ++i)
        {
            response[3 + i] = 0xFF; // All coils = ON
        }
        appendCRC(response, 3 + byteCount);
        break;
    }

    case 0x03: // Read Holding Registers
    case 0x04: // Read Input Registers
    {
        int byteCount = quantity * 2;
        response[2] = byteCount;
        for (int i = 0; i < quantity; ++i)
        {
            response[3 + i * 2] = 0xAB;
            response[4 + i * 2] = 0xCD;
        }
        appendCRC(response, 3 + byteCount);
        break;
    }

    case 0x05: // Write Single Coil
    case 0x06: // Write Single Register
    {
        for (int i = 0; i < 4; ++i)
        {
            response[2 + i] = request[2 + i];
        }
        appendCRC(response, 6);
        break;
    }

    case 0x0F: // Write Multiple Coils
    case 0x10: // Write Multiple Registers
    {
        response[2] = request[2]; // Start address Hi
        response[3] = request[3]; // Start address Lo
        response[4] = request[4]; // Quantity Hi
        response[5] = request[5]; // Quantity Lo
        appendCRC(response, 6);
        break;
    }

    default:
    {
        // Echo function code, return error (optional)
        response[1] = request[1] | 0x80;
        response[2] = 0x01; // Illegal function
        appendCRC(response, 3);
        break;
    }
    }
}

// Send response to master
void sendResponse()
{
    int len = responseLength();
    for (int i = 0; i < len; ++i)
    {
        Serial.write(response[i]);
    }
}

// Append CRC16 to frame
void appendCRC(uint8_t *frame, int len)
{
    uint16_t crc = modbusCRC(frame, len);
    frame[len] = crc & 0xFF;
    frame[len + 1] = crc >> 8;
}

// Calculate Modbus RTU CRC16
uint16_t modbusCRC(uint8_t *buf, int len)
{
    uint16_t crc = 0xFFFF;
    for (int i = 0; i < len; ++i)
    {
        crc ^= buf[i];
        for (int j = 0; j < 8; ++j)
        {
            if (crc & 0x0001)
                crc = (crc >> 1) ^ 0xA001;
            else
                crc = crc >> 1;
        }
    }
    return crc;
}

// Get total response length
int responseLength()
{
    uint8_t func = response[1];
    switch (func)
    {
    case 0x01:
    case 0x02:
    case 0x03:
    case 0x04:
        return 3 + response[2] + 2;
    default:
        return 8;
    }
}
