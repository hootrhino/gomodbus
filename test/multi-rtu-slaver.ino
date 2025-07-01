/*
 * Arduino Modbus Slave for Testing
 * Responds to any valid Modbus RTU request
 * - Even number coils: always 1
 * - Odd number coils: always 0
 * - All other registers: always 0x1234
 */

#include <SoftwareSerial.h>

// Modbus configuration
#define SLAVE_ID 1
#define RS485_ENABLE_PIN 2 // DE/RE pin for RS485 transceiver
#define BAUD_RATE 9600

// Modbus function codes
#define READ_COILS 0x01
#define READ_DISCRETE_INPUTS 0x02
#define READ_HOLDING_REGISTERS 0x03
#define READ_INPUT_REGISTERS 0x04
#define WRITE_SINGLE_COIL 0x05
#define WRITE_SINGLE_REGISTER 0x06
#define WRITE_MULTIPLE_COILS 0x0F
#define WRITE_MULTIPLE_REGISTERS 0x10
#define REPORT_SERVER_ID 0x11

// Buffer for Modbus communication
uint8_t modbusBuffer[256];
uint8_t responseBuffer[256];

void setup()
{
    Serial.begin(BAUD_RATE);
    pinMode(RS485_ENABLE_PIN, OUTPUT);
    digitalWrite(RS485_ENABLE_PIN, LOW); // Receive mode

    Serial.println("Modbus Slave Test Device Ready");
    Serial.print("Slave ID: ");
    Serial.println(SLAVE_ID);
}

void loop()
{
    if (Serial.available())
    {
        int bytesRead = readModbusFrame();
        if (bytesRead > 0)
        {
            processModbusRequest(bytesRead);
        }
    }
}

// Read complete Modbus frame with timeout
int readModbusFrame()
{
    int index = 0;
    unsigned long startTime = millis();

    while (millis() - startTime < 100)
    { // 100ms timeout
        if (Serial.available())
        {
            modbusBuffer[index++] = Serial.read();
            startTime = millis(); // Reset timeout on new byte

            if (index >= 256)
                break; // Prevent buffer overflow
        }
    }

    return index;
}

// Process incoming Modbus request
void processModbusRequest(int frameLength)
{
    if (frameLength < 4)
        return; // Minimum frame length

    uint8_t slaveId = modbusBuffer[0];
    uint8_t functionCode = modbusBuffer[1];

    // Check if request is for this slave
    if (slaveId != SLAVE_ID)
        return;

    // Verify CRC
    if (!verifyCRC(modbusBuffer, frameLength))
    {
        sendErrorResponse(functionCode, 0x04); // Slave device failure
        return;
    }

    // Process based on function code
    switch (functionCode)
    {
    case READ_COILS:
    case READ_DISCRETE_INPUTS:
        handleReadCoils();
        break;

    case READ_HOLDING_REGISTERS:
    case READ_INPUT_REGISTERS:
        handleReadRegisters();
        break;

    case WRITE_SINGLE_COIL:
        handleWriteSingleCoil();
        break;

    case WRITE_SINGLE_REGISTER:
        handleWriteSingleRegister();
        break;

    case WRITE_MULTIPLE_COILS:
        handleWriteMultipleCoils();
        break;

    case WRITE_MULTIPLE_REGISTERS:
        handleWriteMultipleRegisters();
        break;

    case REPORT_SERVER_ID:
        handleReportServerId();
        break;

    default:
        sendErrorResponse(functionCode, 0x01); // Illegal function
        break;
    }
}

// Handle read coils/discrete inputs
void handleReadCoils()
{
    uint16_t startAddress = (modbusBuffer[2] << 8) | modbusBuffer[3];
    uint16_t quantity = (modbusBuffer[4] << 8) | modbusBuffer[5];

    if (quantity == 0 || quantity > 2000)
    {
        sendErrorResponse(modbusBuffer[1], 0x03); // Illegal data value
        return;
    }

    uint8_t byteCount = (quantity + 7) / 8; // Calculate bytes needed

    // Build response
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = modbusBuffer[1]; // Function code
    responseBuffer[2] = byteCount;

    // Fill coil data: even addresses = 1, odd addresses = 0
    for (int i = 0; i < byteCount; i++)
    {
        uint8_t byteValue = 0;
        for (int bit = 0; bit < 8 && (i * 8 + bit) < quantity; bit++)
        {
            uint16_t coilAddress = startAddress + (i * 8 + bit);
            if (coilAddress % 2 == 0)
            { // Even address
                byteValue |= (1 << bit);
            }
            // Odd addresses remain 0 (default)
        }
        responseBuffer[3 + i] = byteValue;
    }

    sendResponse(3 + byteCount);
}

// Handle read holding/input registers
void handleReadRegisters()
{
    uint16_t startAddress = (modbusBuffer[2] << 8) | modbusBuffer[3];
    uint16_t quantity = (modbusBuffer[4] << 8) | modbusBuffer[5];

    if (quantity == 0 || quantity > 125)
    {
        sendErrorResponse(modbusBuffer[1], 0x03); // Illegal data value
        return;
    }

    uint8_t byteCount = quantity * 2;

    // Build response
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = modbusBuffer[1]; // Function code
    responseBuffer[2] = byteCount;

    // Fill register data: all registers = 0x1234
    for (int i = 0; i < quantity; i++)
    {
        responseBuffer[3 + i * 2] = 0x12; // High byte
        responseBuffer[4 + i * 2] = 0x34; // Low byte
    }

    sendResponse(3 + byteCount);
}

// Handle write single coil
void handleWriteSingleCoil()
{
    uint16_t address = (modbusBuffer[2] << 8) | modbusBuffer[3];
    uint16_t value = (modbusBuffer[4] << 8) | modbusBuffer[5];

    // Validate coil value (0x0000 or 0xFF00)
    if (value != 0x0000 && value != 0xFF00)
    {
        sendErrorResponse(modbusBuffer[1], 0x03); // Illegal data value
        return;
    }

    // Echo back the request (successful write confirmation)
    memcpy(responseBuffer, modbusBuffer, 6);
    sendResponse(6);
}

// Handle report server ID (0x11)
void handleReportServerId()
{
    // Server ID information
    const char *serverId = "Arduino Test Slave";
    const char *additionalData = "v1.0";

    uint8_t serverIdLen = strlen(serverId);
    uint8_t additionalLen = strlen(additionalData);
    uint8_t byteCount = serverIdLen + additionalLen + 2; // +2 for run indicator and additional data

    // Build response
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = REPORT_SERVER_ID;
    responseBuffer[2] = byteCount;

    // Server ID
    memcpy(&responseBuffer[3], serverId, serverIdLen);

    // Run Indicator Status (0xFF = ON, 0x00 = OFF)
    responseBuffer[3 + serverIdLen] = 0xFF; // Always running

    // Additional data (optional)
    memcpy(&responseBuffer[4 + serverIdLen], additionalData, additionalLen);

    sendResponse(3 + byteCount);
}

// Handle write single register
void handleWriteSingleRegister()
{
    // Echo back the request (successful write confirmation)
    memcpy(responseBuffer, modbusBuffer, 6);
    sendResponse(6);
}

// Handle write multiple coils
void handleWriteMultipleCoils()
{
    uint16_t startAddress = (modbusBuffer[2] << 8) | modbusBuffer[3];
    uint16_t quantity = (modbusBuffer[4] << 8) | modbusBuffer[5];

    // Build response with address and quantity
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = modbusBuffer[1]; // Function code
    responseBuffer[2] = modbusBuffer[2]; // Start address high
    responseBuffer[3] = modbusBuffer[3]; // Start address low
    responseBuffer[4] = modbusBuffer[4]; // Quantity high
    responseBuffer[5] = modbusBuffer[5]; // Quantity low

    sendResponse(6);
}

// Handle write multiple registers
void handleWriteMultipleRegisters()
{
    uint16_t startAddress = (modbusBuffer[2] << 8) | modbusBuffer[3];
    uint16_t quantity = (modbusBuffer[4] << 8) | modbusBuffer[5];

    // Build response with address and quantity
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = modbusBuffer[1]; // Function code
    responseBuffer[2] = modbusBuffer[2]; // Start address high
    responseBuffer[3] = modbusBuffer[3]; // Start address low
    responseBuffer[4] = modbusBuffer[4]; // Quantity high
    responseBuffer[5] = modbusBuffer[5]; // Quantity low

    sendResponse(6);
}

// Send error response
void sendErrorResponse(uint8_t functionCode, uint8_t exceptionCode)
{
    responseBuffer[0] = SLAVE_ID;
    responseBuffer[1] = functionCode | 0x80; // Set error bit
    responseBuffer[2] = exceptionCode;

    sendResponse(3);
}

// Send response with CRC
void sendResponse(int dataLength)
{
    // Calculate and append CRC
    uint16_t crc = calculateCRC(responseBuffer, dataLength);
    responseBuffer[dataLength] = crc & 0xFF;            // CRC low byte
    responseBuffer[dataLength + 1] = (crc >> 8) & 0xFF; // CRC high byte

    // Switch to transmit mode
    digitalWrite(RS485_ENABLE_PIN, HIGH);
    delayMicroseconds(50);

    // Send response
    Serial.write(responseBuffer, dataLength + 2);
    Serial.flush();

    // Switch back to receive mode
    delayMicroseconds(50);
    digitalWrite(RS485_ENABLE_PIN, LOW);
}

// CRC-16 calculation for Modbus
uint16_t calculateCRC(uint8_t *data, int length)
{
    uint16_t crc = 0xFFFF;

    for (int i = 0; i < length; i++)
    {
        crc ^= data[i];
        for (int j = 0; j < 8; j++)
        {
            if (crc & 0x0001)
            {
                crc = (crc >> 1) ^ 0xA001;
            }
            else
            {
                crc >>= 1;
            }
        }
    }

    return crc;
}

// Verify CRC of received frame
bool verifyCRC(uint8_t *data, int length)
{
    if (length < 4)
        return false;

    uint16_t receivedCRC = data[length - 2] | (data[length - 1] << 8);
    uint16_t calculatedCRC = calculateCRC(data, length - 2);

    return (receivedCRC == calculatedCRC);
}