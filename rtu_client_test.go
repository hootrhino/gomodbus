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
		Timeout:  5000 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	config := RTUConfig{
		Timeout:       1 * time.Second,
		InterCharTime: 3 * time.Millisecond,
		FrameTimeout:  100 * time.Millisecond,
		MaxFrameSize:  256,
	}
	handler := NewModbusRTUHandler(port, config)
	testRTUHandler(t, handler)
}

func testRTUHandler(t *testing.T, handler ModbusApi) {
	// Read holding registers
	registers, err := handler.ReadHoldingRegisters(1, 0, 10)
	if err != nil {
		t.Fatalf("Failed to read holding registers: %v", err)
	}
	t.Logf("Holding Registers: %v", registers)

}
