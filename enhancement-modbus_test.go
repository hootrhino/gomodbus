package modbus

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestDecodeValueAsInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    DeviceRegister
		expect   float64
		expectTy interface{}
	}{
		{
			name: "uint16_ABCD",
			input: DeviceRegister{
				Value:     [4]byte{0x01, 0x02, 0x00, 0x00},
				DataOrder: "ABCD",
				DataType:  "uint16",
			},
			expect:   258,
			expectTy: uint16(258),
		},
		{
			name: "int16_DCBA",
			input: DeviceRegister{
				Value:     [4]byte{0xFF, 0xFE, 0x00, 0x00},
				DataOrder: "DCBA",
				DataType:  "int16",
			},
			expect:   -2,
			expectTy: int16(-2),
		},
		{
			name: "uint32_ABCD",
			input: DeviceRegister{
				Value:     [4]byte{0x00, 0x00, 0x01, 0x00},
				DataOrder: "ABCD",
				DataType:  "uint32",
			},
			expect:   65536,
			expectTy: uint32(65536),
		},
		{
			name: "int32_CDAB",
			input: DeviceRegister{
				Value:     [4]byte{0x00, 0x00, 0xFF, 0xFE},
				DataOrder: "CDAB",
				DataType:  "int32",
			},
			expect:   -2,
			expectTy: int32(-2),
		},
		{
			name: "float32_ABCD",
			input: DeviceRegister{
				Value:     [4]byte{0x42, 0x48, 0x00, 0x00},
				DataOrder: "ABCD",
				DataType:  "float32",
			},
			expect:   50.0,
			expectTy: float32(50.0),
		},
		{
			name: "float64_truncated",
			input: DeviceRegister{
				Value:     [4]byte{0x40, 0x49, 0x0f, 0xdb},
				DataOrder: "ABCD",
				DataType:  "float64",
			},
			expect:   3.141592653589793,
			expectTy: float64(3.141592653589793),
		},
		{
			name: "invalid_type",
			input: DeviceRegister{
				Value:     [4]byte{0x00, 0x00, 0x00, 0x00},
				DataOrder: "ABCD",
				DataType:  "invalid",
			},
			expect:   0,
			expectTy: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := DecodeValueAsInterface(tt.input)
			if tt.expectTy == nil && err == nil {
				t.Errorf("expected error, got nil")
				return
			} else if tt.expectTy != nil && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !FuzzyEqual(val.Float64, tt.expect) {
				t.Errorf("expected float64 %v, got %v", tt.expect, val.Float64)
			}
			t.Logf("== val.AsType: %T", val.AsType)
			switch v := val.AsType.(type) {
			case uint16:
				if v != tt.expectTy.(uint16) {
					t.Errorf("expected uint16 %v, got %v", tt.expectTy, v)
				}
			case int16:
				if v != tt.expectTy.(int16) {
					t.Errorf("expected int16 %v, got %v", tt.expectTy, v)
				}
			case uint32:
				if v != tt.expectTy.(uint32) {
					t.Errorf("expected uint32 %v, got %v", tt.expectTy, v)
				}
			case int32:
				if v != tt.expectTy.(int32) {
					t.Errorf("expected int32 %v, got %v", tt.expectTy, v)
				}
			case float32:
				if !FuzzyEqual(float64(v), float64(tt.expectTy.(float32))) {
					t.Errorf("expected float32 %v, got %v", tt.expectTy, v)
				}
			case float64:
				if !FuzzyEqual(v, tt.expectTy.(float64)) {
					t.Errorf("expected float64 %v, got %v", tt.expectTy, v)
				}
			case nil:
				// expected nil
			default:
				t.Errorf("unexpected type %T", v)
			}
		})
	}
}

// go test -timeout 30s -run ^Test_GroupDeviceRegister$ github.com/hootrhino/gomodbus -v -count=1
func Test_GroupDeviceRegister(t *testing.T) {
	input := []DeviceRegister{
		{Tag: "F", Alias: "A6", SlaverId: 1, Function: 3, Address: 1, Quantity: 1},
		{Tag: "A", Alias: "A1", SlaverId: 1, Function: 3, Address: 2, Quantity: 1},
		{Tag: "B", Alias: "A2", SlaverId: 1, Function: 3, Address: 4, Quantity: 1},
		{Tag: "C", Alias: "A3", SlaverId: 1, Function: 3, Address: 5, Quantity: 1},
		{Tag: "D", Alias: "A4", SlaverId: 1, Function: 3, Address: 8, Quantity: 1},
		{Tag: "E", Alias: "A5", SlaverId: 1, Function: 3, Address: 9, Quantity: 1},
		{Tag: "G", Alias: "A7", SlaverId: 1, Function: 3, Address: 10, Quantity: 1},
	}

	{
		grouped := GroupDeviceRegister(input)
		jsonData, err := json.MarshalIndent(grouped, "", "  ")
		if err != nil {
			t.Fatalf("error marshalling result: %v", err)
		}
		t.Logf("Grouped: %s", string(jsonData))
	}
	handler := NewRTUClientHandler("COM3")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)

	err := handler.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	client := NewClient(handler)
	defer client.GetTransporter().Close()
	result := client.ReadGroupedRegisterValue(input)
	for i, group := range result {
		for j, reg := range group {
			t.Logf("======= group->%v  reg=%v  Address= %v  Tag= %v", i, j, reg.Address, reg.Tag)
		}
	}
}

// go test -timeout 30s -run ^Test_GroupDevice_125_Registers$ github.com/hootrhino/gomodbus -v -count=1
func Test_GroupDevice_125_Registers(t *testing.T) {
	// Group 1: 1-25
	input1 := make([]DeviceRegister, 10)
	for i := 0; i < 10; i++ {
		input1[i].Address = uint16(i + 1)
		input1[i].Tag = fmt.Sprintf("Tag%d", i+1)
		input1[i].Alias = fmt.Sprintf("Alias%d", i+1)
		input1[i].Function = 3
		input1[i].SlaverId = 1
		input1[i].Frequency = 1
		input1[i].Quantity = 1
		input1[i].DataType = "uint16"
		input1[i].DataOrder = "ABCD"
		input1[i].Weight = 1.0
		input1[i].Value = [4]byte{0, 0, 0, 0}
	}
	input2 := make([]DeviceRegister, 10)
	for i := 26; i < 36; i++ {
		input2[i-26].Address = uint16(i + 1)
		input2[i-26].Tag = fmt.Sprintf("Tag%d", i+1)
		input2[i-26].Alias = fmt.Sprintf("Alias%d", i+1)
		input2[i-26].Function = 3
		input2[i-26].SlaverId = 1
		input2[i-26].Frequency = 1
		input2[i-26].Quantity = 1
		input2[i-26].DataType = "uint16"
		input2[i-26].DataOrder = "ABCD"
		input2[i-26].Weight = 1.0
		input2[i-26].Value = [4]byte{0, 0, 0, 0}
	}

	// Group the registers
	handler := NewRTUClientHandler("COM3")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)

	err := handler.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	client := NewClient(handler)
	defer client.GetTransporter().Close()
	testGroup(t, client, input1)
	testGroup(t, client, input2)
}
func testGroup(t *testing.T, client Client, input []DeviceRegister) {
	result := client.ReadGroupedRegisterValue(input)
	for i, group := range result {
		for j, reg := range group {
			decodeValue, err := reg.DecodeValueAsInterface()
			if err != nil {
				t.Errorf("Error decoding value: %v", err)
			}
			t.Logf("== Group[%v]\nIndex= %v\nAddress= %v\nTag= %v\nValue = %v\nDataType= %v\nDataOrder= %v\ndecodeValue= %v\n",
				i, j, reg.Address, reg.Tag, reg.Value, reg.DataType, reg.DataOrder, decodeValue.Float64)
		}
	}
}
