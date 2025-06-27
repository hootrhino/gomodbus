package modbus

import (
	"testing"
	"time"

	serial "github.com/hootrhino/goserial"
)

func TestModbusSlaverRTU(t *testing.T) {
	port, err := serial.Open(&serial.Config{
		Address:  "COM3",
		BaudRate: 9600,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  5000 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	handler := NewModbusRTUHandler(port, 1*time.Second)
	testRTUHandler(t, handler)
}

func testRTUHandler(t *testing.T, handler ModbusApi) {

}
