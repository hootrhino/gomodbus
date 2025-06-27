package modbus

import (
	"net"
	"testing"
	"time"

	"log"
	"os"

	modbus_server "github.com/hootrhino/mbserver"
	"github.com/hootrhino/mbserver/store"
)

// startTestTCPServer initializes a Modbus TCP server with sample holding registers.
func startTestTCPServer() *modbus_server.Server {

	// Initialize a Modbus server
	server := modbus_server.NewServer(store.NewInMemoryStore(), 1)

	// Set an error handler
	server.SetErrorHandler(func(err error) {
		log.Printf("Modbus server error: %v", err)
	})

	// Set up logger
	server.SetLogger(os.Stdout)

	// Set more sample holding register data
	sampleHoldingRegisters := make([]uint16, 10) // Adjust size as needed
	for i := range sampleHoldingRegisters {
		sampleHoldingRegisters[i] = 0xABCD // Sample data for holding registers
	}
	if err := server.SetHoldingRegisters(sampleHoldingRegisters); err != nil {
		log.Fatalf("Failed to set holding registers: %v", err)
	}

	// Start the Modbus server
	log.Println("Starting Modbus server on :502")
	if err := server.Start(":502"); err != nil {
		log.Fatalf("Failed to start Modbus server: %v", err)
	}
	return server
}

func TestModbusSlaverTCP(t *testing.T) {
	// server := startTestTCPServer()
	// defer server.Stop()
	conn, err := net.Dial("tcp", "localhost:502")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	handler := NewModbusTCPHandler(conn, 5*time.Second)
	testTCPHandler(t, handler)

}

func testTCPHandler(t *testing.T, handler ModbusApi) {

	for i := range 2 {
		result1, err := handler.ReadInputRegisters(1, uint16(i), 1)
		if err != nil {
			t.Fatalf("ReadInputRegisters failed: %v", err)
		}
		t.Log("ReadInputRegisters=", result1)
		assertUint16Equal(t, []uint16{0xABCD}, result1)
	}

}
