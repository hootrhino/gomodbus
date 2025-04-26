package modbus

import (
	"testing"
)

func TestBuildRequestPDU(t *testing.T) {
	functionCode := uint8(0x03)
	data := []byte{0x00, 0x0A, 0x00, 0x01}
	expectedPDU := []byte{0x03, 0x00, 0x0A, 0x00, 0x01}

	pdu, err := buildRequestPDU(functionCode, data)
	if err != nil {
		t.Fatalf("BuildRequestPDU failed: %v", err)
	}
	if !equal(pdu, expectedPDU) {
		t.Errorf("BuildRequestPDU returned incorrect PDU: got %v, expected %v", pdu, expectedPDU)
	}
}

func TestGetExceptionMessage(t *testing.T) {
	testCases := []struct {
		code    uint8
		message string
	}{
		{code: 0x01, message: "Illegal function"},
		{code: 0x02, message: "Illegal data address"},
		{code: 0x03, message: "Illegal data value"},
		{code: 0x04, message: "Slave device failure"},
		{code: 0x05, message: "Acknowledge"},
		{code: 0x06, message: "Slave device busy"},
		{code: 0x08, message: "Memory parity error"},
		{code: 0x0A, message: "Gateway path unavailable"},
		{code: 0x0B, message: "Gateway target device failed to respond"},
		{code: 0xFF, message: "Unknown exception code"},
	}

	for _, tc := range testCases {
		message := getExceptionMessage(tc.code)
		if message != tc.message {
			t.Errorf("GetExceptionMessage(%#02x) returned incorrect message: got %q, expected %q", tc.code, message, tc.message)
		}
	}
}

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

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
