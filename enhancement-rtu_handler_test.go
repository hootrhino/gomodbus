package modbus

import (
	"bytes"
	"os"
	"testing"
	"time"

	serial "github.com/hootrhino/goserial"
)

// Mock Serial Port
type mockSerialPort struct {
	readBuffer  bytes.Buffer
	writeBuffer bytes.Buffer
	closed      bool
}

func (m *mockSerialPort) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockSerialPort) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *mockSerialPort) Close() error {
	m.closed = true
	return nil
}

func TestRTUHandler_ReadCoils(t *testing.T) {
	// Mock response for Read Coils (0x01)
	// Slave ID (1) + Function Code (1) + Byte Count (1) + Data (1 byte) + CRC (2)
	mockResponse := bytes.NewBuffer([]byte{
		0x01,       // Slave ID
		0x01,       // Function Code
		0x01,       // Byte Count
		0x05,       // Coil Status
		0x84, 0x0A, // CRC
	})
	mockPort := &mockSerialPort{readBuffer: *mockResponse}
	handler := NewRTUHandler(mockPort, 1*time.Second, os.Stdout)

	coils, err := handler.ReadCoils(1, 0, 3)
	if err != nil {
		t.Fatalf("ReadCoils failed: %v", err)
	}
	expectedCoils := []bool{true, false, true}
	if !equalBool(coils, expectedCoils) {
		t.Errorf("ReadCoils returned incorrect values: got %v, expected %v", coils, expectedCoils)
	}

	// Check sent request PDU (basic check)
	expectedRequest := []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x03, 0xF8, 0xF0} // Slave ID + PDU + CRC
	sent := mockPort.writeBuffer.Bytes()
	if !equal(sent, expectedRequest) {
		t.Errorf("ReadCoils sent incorrect request: got %v, expected %v", sent, expectedRequest)
	}
}

// Similar test functions for other RTUHandler methods (ReadDiscreteInputs, ReadHoldingRegisters, etc.)

func TestRTUHandler_WriteSingleCoil(t *testing.T) {
	// Mock response for Write Single Coil (0x05) - echo of request
	mockResponse := bytes.NewBuffer([]byte{
		0x01,       // Slave ID
		0x05,       // Function Code
		0x00, 0x0A, // Address
		0xFF, 0x00, // Value (ON)
		0x4F, 0x8A, // CRC
	})
	mockPort := &mockSerialPort{readBuffer: *mockResponse}
	handler := NewRTUHandler(mockPort, 1*time.Second, os.Stdout)

	err := handler.WriteSingleCoil(1, 10, true)
	if err != nil {
		t.Fatalf("WriteSingleCoil failed: %v", err)
	}

	// Check sent request
	expectedRequest := []byte{0x01, 0x05, 0x00, 0x0A, 0xFF, 0x00, 0x15, 0x05}
	sent := mockPort.writeBuffer.Bytes()
	if !equal(sent, expectedRequest) {
		t.Errorf("WriteSingleCoil sent incorrect request: got %v, expected %v", sent, expectedRequest)
	}
}

// ... (Similar test functions for other write methods)

func TestRTUHandler_SendAndReceive_Error(t *testing.T) {
	mockPort := &mockSerialPort{readBuffer: *bytes.NewBuffer([]byte{})} // Empty buffer for immediate read error
	handler := NewRTUHandler(mockPort, 1*time.Second, os.Stdout)

	_, err := handler.ReadCoils(1, 0, 1)
	if err == nil {
		t.Fatalf("Expected error during sendAndReceive")
	}
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
	handler := NewRTUHandler(port, 1*time.Second, os.Stdout)
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
			result1, err := handler.ReadHoldingRegisters(1, 0, 5)
			if err != nil {
				t.Fatalf("ReadHoldingRegisters failed: %v", err)
			}
			t.Log("ReadHoldingRegisters=", result1)
		}
	}
	{
		for i := 0; i < 10; i++ {
			result1, err := handler.ReadBytes(1, 0, 20, "BIG")
			if err != nil {
				t.Fatalf("ReadBytes failed: %v", err)
			}
			t.Log("ReadBytes=", result1)
		}
	}

}
