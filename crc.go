package modbus

// CRC16_MODBUS_POLYNOMIAL is the polynomial for Modbus CRC16: x^16 + x^15 + x^2 + 1 (0x8005 in normal, 0xA001 in reverse)
const CRC16_MODBUS_POLYNOMIAL uint16 = 0xA001 // This is the reverse polynomial of 0x8005
const CRC16_MODBUS_INITIAL_VALUE uint16 = 0xFFFF

// CRC16 calculates the Modbus CRC16 checksum for the given data.
// It uses a polynomial of 0xA001 (reversed 0x8005) and an initial value of 0xFFFF.
// The result is the 16-bit CRC.
func CRC16(data []byte) uint16 {
	var crc uint16 = CRC16_MODBUS_INITIAL_VALUE

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ CRC16_MODBUS_POLYNOMIAL
			} else {
				crc = crc >> 1
			}
		}
	}
	return crc
}
