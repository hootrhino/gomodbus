package modbus

import "testing"

func TestCRC16(t *testing.T) {
	testCases := []struct {
		data     []byte
		expected uint16
	}{
		{data: []byte{0x01, 0x03, 0x02, 0x12, 0x34}, expected: 0x33B5},
		{data: []byte{01, 03, 00, 00, 00, 01}, expected: 0x0A84},
		{data: []byte{01, 03, 14, 12, 34, 12, 34, 12, 34,
			12, 34, 12, 34, 12, 34, 12, 34, 12, 34, 12, 34, 12, 34}, expected: 0x0C7D},
		{data: []byte{}, expected: 0xFFFF},     // Empty data, CRC should be initial value
		{data: []byte{0x00}, expected: 0x40BF}, // Single zero byte
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
