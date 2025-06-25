package modbus

import (
	"testing"
	"time"

	"net"

	serial "github.com/hootrhino/goserial"
)

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
