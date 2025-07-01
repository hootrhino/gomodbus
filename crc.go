package modbus

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

// CRC16Swap big-endian CRC16 calculation.
// This function is not implemented yet, but it should calculate the CRC16 in big-endian
// order, which is useful for certain Modbus applications that require the CRC to be in big-endian format.
func CRCBigEndian(data []byte) uint16 {
	crc := CRC16(data)
	return CRC16Swap(crc)
}

// CRC16Swap swaps the bytes of a CRC16 value.
func CRC16Swap(crc uint16) uint16 {
	return (crc >> 8) | (crc << 8)
}
