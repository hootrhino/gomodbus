package modbus

import (
	"testing"
	"time"

	serial "github.com/hootrhino/goserial"
)
import "net"

func TestModbusSlaverTCP(t *testing.T) {
	// Import the net package to fix the undefined error
	conn, err := net.Dial("tcp", "localhost:502")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	handler := NewModbusTCPHandler(conn, 1*time.Second)
	testHandler(t, handler)

}
func TestModbusSlaverRTU(t *testing.T) {
	port, err := serial.Open(&serial.Config{
		Address:  "COM3",
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
	handler := NewModbusRTUHandler(port, 1*time.Second)
	testHandler(t, handler)
}

func testHandler(t *testing.T, handler ModbusApi) {
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadCoils(1, 0, 3)
			if err != nil {
				t.Fatalf("ReadCoils failed: %v", err)
			}
			t.Log("ReadCoils=", result1)
		}
	}
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadDiscreteInputs(1, 0, 3)
			if err != nil {
				t.Fatalf("ReadDiscreteInputs failed: %v", err)
			}
			t.Log("ReadDiscreteInputs=", result1)
		}
	}
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadHoldingRegisters(1, 0, 5)
			if err != nil {
				t.Fatalf("ReadHoldingRegisters failed: %v", err)
			}
			t.Log("ReadHoldingRegisters=", result1)
		}
	}
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadInputRegisters(1, 0, 5)
			if err != nil {
				t.Fatalf("ReadInputRegisters failed: %v", err)
			}
			t.Log("ReadInputRegisters=", result1)
		}
	}
	{
		for i := 0; i < 10; i++ {
			err := handler.WriteSingleCoil(1, 0, true)
			if err != nil {
				t.Fatalf("WriteSingleCoil failed: %v", err)
			}
		}
	}
	{
		for i := 0; i < 10; i++ {
			err := handler.WriteSingleRegister(1, 0, 100)
			if err != nil {
				t.Fatalf("WriteSingleRegister failed: %v", err)
			}
		}
	}
	{
		for i := 0; i < 10; i++ {
			err := handler.WriteMultipleCoils(1, 0, []bool{true, false, true})
			if err != nil {
				t.Fatalf("WriteMultipleCoils failed: %v", err)
			}
		}
	}
	{
		for i := 0; i < 10; i++ {
			err := handler.WriteMultipleRegisters(1, 0, []uint16{1, 2, 3})
			if err != nil {
				t.Fatalf("WriteMultipleRegisters failed: %v", err)
			}
		}
	}
}
