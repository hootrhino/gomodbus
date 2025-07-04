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
func StartTestTCPServer() *modbus_server.Server {

	// Create an in-memory store instance
	memStore := store.NewInMemoryStore().(*store.InMemoryStore)
	// Set sample holding register data
	defaultHoldingRegistersSize := 10
	memStore.SetHoldingRegisters(make([]uint16, defaultHoldingRegistersSize))

	// Set maximum concurrent connections
	maxConns := 100
	// Initialize a Modbus server
	server := modbus_server.NewServer(memStore, maxConns)

	// Set an error handler
	server.SetErrorHandler(func(err error) {
		log.Printf("Modbus server error: %v", err)
	})

	// Set up logger
	server.SetLogger(os.Stdout)

	// Set more sample holding register data
	sampleHoldingRegisters := make([]uint16, 10)
	for i := range sampleHoldingRegisters {
		sampleHoldingRegisters[i] = 0xABCD
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
	server := StartTestTCPServer()
	defer server.Stop()
	conn, err := net.Dial("tcp", "localhost:502")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	handler := NewModbusTCPHandler(conn, TCPTransporterConfig{
		Timeout:    5 * time.Second,
		RetryDelay: 200 * time.Millisecond,
	})
	testTCPHandler(t, handler)

}

func testTCPHandler(t *testing.T, handler ModbusApi) {
	for i := range 9 {
		result1, err := handler.ReadHoldingRegisters(1, uint16(i), 1)
		if err != nil {
			t.Fatalf("ReadHoldingRegisters failed: %v", err)
		}
		t.Logf("ReadHoldingRegisters result: %X", result1)
		if err := AssertUint16Equal([]uint16{0xABCD}, result1); err != nil {
			t.Fatalf("ReadHoldingRegisters result mismatch: %v", err)
		}
	}

}
