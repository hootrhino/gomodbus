package modbus

import (
	"net"
	"testing"
	"time"
)

func TestRegisterManager_LoadRegisters(t *testing.T) {
	server := StartTestTCPServer()
	defer server.Stop()
	conn, err1 := net.Dial("tcp", "localhost:502")
	if err1 != nil {
		t.Fatalf("Failed to connect to server: %v", err1)
	}
	defer conn.Close()
	handler := NewModbusTCPHandler(conn, TCPTransporterConfig{
		Timeout:    5 * time.Second,
		RetryDelay: 200 * time.Millisecond,
	})
	manager := NewRegisterManager(handler, 10)
	manager.OnErrorCallback = func(err error) {
		t.Errorf("OnErrorCallback() error = %v, want nil", err)
	}
	manager.OnReadCallback = func(registers []DeviceRegister) {
		for _, reg := range registers {
			reg.DecodeValue()
			DecodeValue, err := reg.DecodeValue()
			if err != nil {
				t.Errorf("DecodeValue() error = %v, want nil", err)
			}
			t.Logf("OnReadCallback() register = %v, value = %v", reg.Tag, DecodeValue)
			if reg.Tag == "tag1" || reg.Tag == "tag2" || reg.Tag == "tag3" || reg.Tag == "tag4" || reg.Tag == "tag5" {
				errA := AssertUint16Equal([]uint16{0xABCD}, []uint16{DecodeValue.AsType.(uint16)})
				if errA != nil {
					t.Fatalf("AssertUint16Equal() error = %v, want nil", errA)
				}
			}
			if reg.Tag == "tag-array-1" {
				errA := AssertUint16Equal([]uint16{0xABCD, 0xABCD, 0xABCD, 0xABCD, 0xABCD},
					DecodeValue.AsType.([]uint16))
				if errA != nil {
					t.Fatalf("AssertUint16Equal() error = %v, want nil", errA)
				}
			}
		}
	}
	// Test loading registers without duplicates
	registers := []DeviceRegister{
		{
			Tag:          "tag1",
			Alias:        "alias1",
			Function:     3,
			ReadAddress:  0,
			ReadQuantity: 1,
			SlaverId:     1,
			DataType:     "uint16",
			DataOrder:    "AB",
		},
		{
			Tag:          "tag2",
			Alias:        "alias2",
			Function:     3,
			ReadAddress:  0,
			ReadQuantity: 1,
			SlaverId:     1,
			DataType:     "uint16",
			DataOrder:    "AB",
		},
		{
			Tag:          "tag3",
			Alias:        "alias3",
			Function:     3,
			ReadAddress:  1,
			ReadQuantity: 1,
			SlaverId:     1,
			DataType:     "uint16",
			DataOrder:    "AB",
		},
		{
			Tag:          "tag4",
			Alias:        "alias4",
			Function:     3,
			ReadAddress:  2,
			ReadQuantity: 1,
			SlaverId:     1,
			DataType:     "uint16",
			DataOrder:    "AB",
		},
		{
			Tag:          "tag5",
			Alias:        "alias5",
			Function:     3,
			ReadAddress:  3,
			ReadQuantity: 1,
			SlaverId:     1,
			DataType:     "uint16",
			DataOrder:    "AB",
		},
		{
			Tag:          "tag-array-1",
			Alias:        "alias-array-1",
			Function:     3,
			ReadAddress:  0,
			ReadQuantity: 5,
			SlaverId:     1,
			DataType:     "uint16[5]",
			DataOrder:    "ABCD",
		},
	}
	err := manager.LoadRegisters(registers)
	if err != nil {
		t.Errorf("LoadRegisters() error = %v, want nil", err)
	}
	manager.Start()
	for i := 0; i < 100; i++ {
		manager.ReadGroupedData()
		time.Sleep(100 * time.Millisecond)
	}
	manager.Stop()
}
