package modbus

import (
	"testing"
)

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
