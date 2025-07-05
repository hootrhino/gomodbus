package modbus

import (
	"testing"
	"time"

	serial "github.com/hootrhino/goserial"
)

func TestModbusSlaverRTU(t *testing.T) {
	port, err := serial.Open(&serial.Config{
		Address:  "COM6",
		BaudRate: 9600,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	config := RTUConfig{
		MaxFrameSize: 256,
	}
	handler := NewModbusRTUHandler(port, config)
	testRTUHandler(t, handler)
}

func testRTUHandler(t *testing.T, handler ModbusApi) {
	for i := 0; i < 10; i++ {
		registers, err := handler.ReadHoldingRegisters(1, 0, 10)
		if err != nil {
			t.Fatalf("Failed to read holding registers: %v", err)
		}
		t.Logf("Holding Registers: %v", registers)
		AssertUint16Equal([]uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, registers)
	}
}
