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
		Timeout:  300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()
	handler := NewModbusRTUHandler(port, 1*time.Second)
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadCoils(1, 0, 3)
			if err != nil {
				t.Fatalf("ReadCoils failed: %v", err)
			}
			expectedResult := []bool{true, true, true}
			if !equalBool(result1, expectedResult) {
				t.Errorf("ReadCoils returned incorrect values: got %v, expected %v", result1, expectedResult)
			}
		}
	}
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadDiscreteInputs(1, 0, 3)
			if err != nil {
				t.Fatalf("ReadDiscreteInputs failed: %v", err)
			}
			expectedResult := []bool{true, true, true}
			if !equalBool(result1, expectedResult) {
				t.Errorf("ReadDiscreteInputs returned incorrect values: got %v, expected %v", result1, expectedResult)
			}
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
}

func equalBool(result1 []bool, expectedResult []bool) bool {
	if len(result1) != len(expectedResult) {
		return false
	}
	for i := 0; i < len(result1); i++ {
		if result1[i] != expectedResult[i] {
			return false
		}
	}
	return true
}
