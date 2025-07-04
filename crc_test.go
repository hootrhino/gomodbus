package modbus

import (
	"testing"
)

func TestCRC(t *testing.T) {
	var crc1 crc
	crc1.reset()
	crc1.pushBytes([]byte{0x02, 0x07})

	if crc1.value() != 0x1241 {
		t.Fatalf("crc expected %v, actual %v", 0x1241, crc1.value())
	}
}
