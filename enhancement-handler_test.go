package modbus

import (
	"fmt"
	"testing"
	"time"

	"net"

	serial "github.com/hootrhino/goserial"
)

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
		Timeout:  5000 * time.Millisecond,
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
		for i := 1; i < 10; i++ {
			result1, err := handler.ReadCoils(uint16(i), 0, 3)
			if err != nil {
				t.Fatalf("ReadCoils failed: %v", err)
			}
			t.Log("ReadCoils=", result1)
			assertBoolsEqual(t, []bool{true, true, true}, result1)
		}
	}
	{
		for i := 1; i < 10; i++ {
			result1, err := handler.ReadDiscreteInputs(uint16(i), 0, 3)
			if err != nil {
				t.Fatalf("ReadDiscreteInputs failed: %v", err)
			}
			t.Log("ReadDiscreteInputs=", result1)
			assertBoolsEqual(t, []bool{true, true, true}, result1)

		}
	}
	{
		for i := 1; i < 10; i++ {
			result1, err := handler.ReadHoldingRegisters(uint16(i), 0, 5)
			if err != nil {
				t.Fatalf("ReadHoldingRegisters failed: %v", err)
			}
			t.Log("ReadHoldingRegisters=", result1)
		}
	}
	{
		for i := 1; i < 10; i++ {
			result1, err := handler.ReadInputRegisters(uint16(i), 0, 5)
			if err != nil {
				t.Fatalf("ReadInputRegisters failed: %v", err)
			}
			t.Log("ReadInputRegisters=", result1)
			assertUint16Equal(t, []uint16{0xABCD, 0xABCD, 0xABCD, 0xABCD, 0xABCD}, result1)
		}
	}
	{
		for i := 1; i < 10; i++ {
			err := handler.WriteSingleCoil(uint16(i), 0, true)
			if err != nil {
				t.Fatalf("WriteSingleCoil failed: %v", err)
			}
		}
	}
	{
		for i := 1; i < 10; i++ {
			err := handler.WriteSingleRegister(uint16(i), 0, 100)
			if err != nil {
				t.Fatalf("WriteSingleRegister failed: %v", err)
			}
		}
	}
	{
		for i := 1; i < 10; i++ {
			err := handler.WriteMultipleCoils(uint16(i), 0, []bool{false, false, false})
			if err != nil {
				t.Fatalf("WriteMultipleCoils failed: %v", err)
			}
		}
	}
	{
		for i := 1; i < 10; i++ {
			err := handler.WriteMultipleRegisters(uint16(i), 0, []uint16{1, 2, 3})
			if err != nil {
				t.Fatalf("WriteMultipleRegisters failed: %v", err)
			}
		}
	}
}
func assertBoolsEqual(t *testing.T, expected []bool, actual []bool) {
	if len(expected) != len(actual) {
		t.Errorf("Expected length %d, but got %d", len(expected), len(actual))
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("Expected %v, but got %v", expected, actual)
			return
		}
	}
}
func assertUint16Equal(t *testing.T, expected []uint16, actual []uint16) {
	if len(expected) != len(actual) {
		t.Errorf("Expected length %d, but got %d", len(expected), len(actual))
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("Expected %v, but got %v", expected, actual)
		}
	}
}

func TestReadRawDeviceIdentity(t *testing.T) {
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
	resp, err := handler.ReadRawDeviceIdentity(1)
	if err != nil {
		t.Fatalf("ReadRawDeviceIdentity failed: %v", err)
	}

	// Validate the response length and content
	if len(resp) < 2 || resp[0] != 0x11 {
		t.Fatalf("Unexpected response: %v", resp)
	}

	t.Logf("Received raw response: %v", resp)
}

func TestReadDeviceIdentityWithHandler(t *testing.T) {

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
	// Define a custom callback to handle the raw response
	customHandler := func(data []byte) error {
		if len(data) < 2 || data[0] != 0x11 {
			return fmt.Errorf("invalid response data: %v", data)
		}
		t.Logf("Parsed data: %v", data)
		return nil
	}

	err1 := handler.ReadDeviceIdentityWithHandler(1, customHandler)
	if err1 != nil {
		t.Fatalf("ReadDeviceIdentityWithHandler failed: %v", err)
	}
}
