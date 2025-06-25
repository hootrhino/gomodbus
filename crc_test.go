package modbus

import "testing"

func TestCRC16(t *testing.T) {
	testCases := []struct {
		data     []byte
		expected uint16
	}{
		{data: []byte{0x01, 0x03, 0x00, 0x0A, 0x00, 0x01}, expected: 0xa408},
		{data: []byte{0x01, 0x04, 0x00, 0x01, 0x00, 0x01}, expected: 0x600a},
		{data: []byte{0x10, 0x06, 0x00, 0x01, 0x00, 0x01, 0x01, 0x08}, expected: 0x4b51},
	}

	for _, tc := range testCases {
		crc := CRC16(tc.data)
		if crc != tc.expected {
			t.Errorf("CRC16(%v) returned incorrect CRC: got %#04x, expected %#04x", tc.data, crc, tc.expected)
		}
	}
}
func TestCRC16WithInvalidData(t *testing.T) {
	testCases := []struct {
		data     []byte
		expected uint16
	}{
		{data: []byte{}, expected: 0xffff},     // Empty data should return 0xffff
		{data: []byte{0x00}, expected: 0x0000}, // Single byte data
	}

	for _, tc := range testCases {
		crc := CRC16(tc.data)
		if crc != tc.expected {
			t.Errorf("CRC16(%v) returned incorrect CRC: got %#04x, expected %#04x", tc.data, crc, tc.expected)
		}
	}
}
