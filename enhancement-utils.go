package modbus

// buildRequestPDU constructs a Modbus request PDU.
// It takes the function code and the data payload as input.
func buildRequestPDU(functionCode uint8, data []byte) ([]byte, error) {
	pdu := make([]byte, 1+len(data))
	pdu[0] = functionCode
	copy(pdu[1:], data)
	return pdu, nil
}

// getExceptionMessage returns a human-readable message for a Modbus exception code.
func getExceptionMessage(exceptionCode uint8) string {
	switch exceptionCode {
	case 0x01:
		return "Illegal function"
	case 0x02:
		return "Illegal data address"
	case 0x03:
		return "Illegal data value"
	case 0x04:
		return "Slave device failure"
	case 0x05:
		return "Acknowledge"
	case 0x06:
		return "Slave device busy"
	case 0x08:
		return "Memory parity error"
	case 0x0A:
		return "Gateway path unavailable"
	case 0x0B:
		return "Gateway target device failed to respond"
	default:
		return "Unknown exception code"
	}
}

// CRC16 calculates the Modbus CRC16 checksum.
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return ((crc & 0xFF) << 8) | ((crc >> 8) & 0xFF)
}
