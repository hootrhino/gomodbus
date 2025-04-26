package modbus

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// Mock TCP connection
type mockTCPConn struct {
	readBuffer  bytes.Buffer
	writeBuffer bytes.Buffer
	closed      bool
}

func (m *mockTCPConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockTCPConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *mockTCPConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockTCPConn) LocalAddr() net.Addr                { return nil }
func (m *mockTCPConn) RemoteAddr() net.Addr               { return nil }
func (m *mockTCPConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockTCPConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockTCPConn) SetWriteDeadline(t time.Time) error { return nil }

func TestTCPHandler_ReadCoils(t *testing.T) {
	// Mock response for Read Coils (0x01)
	// MBAP (7 bytes) + Function Code (1) + Byte Count (1) + Data (1 byte for 8 coils)
	mockResponse := bytes.NewBuffer([]byte{
		0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0x01, // MBAP
		0x01, // Function Code
		0x01, // Byte Count
		0x05, // Coil Status (00000101 - Coils 1 and 3 are ON)
	})
	mockConn := &mockTCPConn{readBuffer: *mockResponse}
	handler := NewTCPHandler(mockConn, 1*time.Second, os.Stdout)

	coils, err := handler.ReadCoils(1, 0, 3)
	if err != nil {
		t.Fatalf("ReadCoils failed: %v", err)
	}
	expectedCoils := []bool{true, false, true}
	if !equalBool(coils, expectedCoils) {
		t.Errorf("ReadCoils returned incorrect values: got %v, expected %v", coils, expectedCoils)
	}

	// Check sent request PDU (basic check)
	// expectedRequestPDU := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x04, 0x01, 0x01, 0x00, 0x00, 0x00, 0x03} // Basic MBAP + PDU
	sent := mockConn.writeBuffer.Bytes()
	if len(sent) < 12 || sent[7] != 0x01 || binary.BigEndian.Uint16(sent[8:10]) != 0 || binary.BigEndian.Uint16(sent[10:12]) != 3 {
		t.Errorf("ReadCoils sent incorrect request: got %v, expected to contain function code 0x01 and address/quantity", sent)
	}
	// Check MBAP header
	if len(sent) < 7 {
		t.Errorf("ReadCoils sent incorrect MBAP header: got %v, expected at least 7 bytes", sent)
	}
}

// Similar test functions for other TCPHandler methods (ReadDiscreteInputs, ReadHoldingRegisters, etc.)
// You would mock the responses and verify the sent requests for each function code.

func TestTCPHandler_WriteSingleCoil(t *testing.T) {
	// Mock response for Write Single Coil (0x05) - echo of request
	mockResponse := bytes.NewBuffer([]byte{
		0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, // MBAP
		0x05,       // Function Code
		0x00, 0x0A, // Address
		0xFF, 0x00, // Value (ON)
	})
	mockConn := &mockTCPConn{readBuffer: *mockResponse}
	handler := NewTCPHandler(mockConn, 1*time.Second, os.Stdout)

	err := handler.WriteSingleCoil(1, 10, true)
	if err != nil {
		t.Fatalf("WriteSingleCoil failed: %v", err)
	}

	// Check sent request PDU
	expectedRequestPDU := []byte{0x01, 0x05, 0x00, 0x00, 0x00, 0x06, 0x01, 0x05, 0x00, 0x0A, 0xFF, 0x00}
	sent := mockConn.writeBuffer.Bytes()
	if !equal(sent, expectedRequestPDU) {
		t.Errorf("WriteSingleCoil sent incorrect request: got %v, expected %v", sent, expectedRequestPDU)
	}
}

// ... (Similar test functions for other write methods)

func TestTCPHandler_SendAndReceive_Error(t *testing.T) {
	mockConn := &mockTCPConn{readBuffer: *bytes.NewBuffer([]byte{})} // Empty buffer for immediate read error
	handler := NewTCPHandler(mockConn, 1*time.Second, os.Stdout)

	_, err := handler.ReadCoils(1, 0, 1)
	if err == nil {
		t.Fatalf("Expected error during sendAndReceive")
	}
}

func TestTCPHandler_SendAndReceive_Exception(t *testing.T) {
	// Mock response with exception (function code + 0x80 | exception code)
	mockResponse := bytes.NewBuffer([]byte{
		0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0x01, // MBAP
		0x81, // Function Code + Error Flag
		0x02, // Exception Code (Illegal Data Address)
	})
	mockConn := &mockTCPConn{readBuffer: *mockResponse}
	handler := NewTCPHandler(mockConn, 1*time.Second, os.Stdout)

	_, err := handler.ReadCoils(1, 0, 1)
	if err == nil {
		t.Fatalf("Expected Modbus exception")
	}
	if !strings.Contains(err.Error(), "Illegal data address") {
		t.Errorf("Expected 'Illegal data address' exception, got: %v", err)
	}
}

func equalBool(a, b []bool) bool {
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

func TestModbusSlaverTCP(t *testing.T) {
	server := "localhost:5020"
	conn, err := net.Dial("tcp", server)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	handler := NewTCPHandler(conn, 1*time.Second, os.Stdout)
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
