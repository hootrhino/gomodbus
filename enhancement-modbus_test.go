package modbus

import (
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"testing"
)

func Test_float32FromBits(t *testing.T) {
	tests := []struct {
		name   string
		bits   uint32
		expect float32
	}{
		{
			name:   "positive_float",
			bits:   0x42480000, // 50.0 in IEEE 754
			expect: 50.0,
		},
		{
			name:   "negative_float",
			bits:   0xC2480000, // -50.0 in IEEE 754
			expect: -50.0,
		},
		{
			name:   "zero",
			bits:   0x00000000, // 0.0 in IEEE 754
			expect: 0.0,
		},
		{
			name:   "small_float",
			bits:   0x3DCCCCCD, // 0.1 in IEEE 754
			expect: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := float32FromBits(tt.bits)
			t.Log("== float32FromBits", result)
			if math.Abs(float64(result-tt.expect)) > 0.0001 {
				t.Errorf("float32FromBits(%#x) = %f, want %f", tt.bits, result, tt.expect)
			}
		})
	}
}

func Test_float64FromBits(t *testing.T) {
	tests := []struct {
		name   string
		bits   uint64
		expect float64
	}{
		{
			name:   "positive_float",
			bits:   0x4049000000000000, // 50.0 in IEEE 754
			expect: 50.0,
		},
		{
			name:   "negative_float",
			bits:   0xC049000000000000, // -50.0 in IEEE 754
			expect: -50.0,
		},
		{
			name:   "zero",
			bits:   0x0000000000000000, // 0.0 in IEEE 754
			expect: 0.0,
		},
		{
			name:   "pi",
			bits:   0x400921FB54442D18, // 3.141592653589793 in IEEE 754
			expect: 3.141592653589793,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := float64FromBits(tt.bits)
			t.Log("== float64FromBits", result)
			if math.Abs(result-tt.expect) > 0.0000001 {
				t.Errorf("float64FromBits(%#x) = %f, want %f", tt.bits, result, tt.expect)
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

// go test -timeout 30s -run ^Test_GroupDevice_UART_125_Registers$ github.com/hootrhino/gomodbus -v -count=1
func Test_GroupDevice_UART_125_Registers(t *testing.T) {
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
		input1[i].Value = [8]byte{0, 0, 0, 0}
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
		input2[i-26].Value = [8]byte{0, 0, 0, 0}
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
func Test_Group_UART_Device_1_Bool_Register(t *testing.T) {
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
	input1 := make([]DeviceRegister, 1)
	for i := 0; i < 16; i++ {
		t.Log("======= Test_Group_UART_Device_1_Bool_Register", i)
		input1[0].Address = uint16(4) // 0000 0000 0000 0100
		input1[0].Tag = "Tag1"
		input1[0].Alias = "Alias1"
		input1[0].Function = 3
		input1[0].SlaverId = 1
		input1[0].Frequency = 1
		input1[0].Quantity = 1
		input1[0].DataType = "bool"
		input1[0].DataOrder = "A"
		input1[0].Weight = 1.0
		input1[0].Value = [8]byte{0, 0, 0, 0}
		input1[0].BitMask = uint16(i)
		testGroup(t, client, input1)
	}

}

// go test -timeout 30s -run ^Test_Group_TCP_Device_125_Registers$ github.com/hootrhino/gomodbus -v -count=1
func Test_Group_TCP_Device_125_Registers(t *testing.T) {
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
		input1[i].Value = [8]byte{0, 0, 0, 0}
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
		input2[i-26].Value = [8]byte{0, 0, 0, 0}
	}

	// Group the registers
	handler := NewTCPClientHandler("127.0.0.1:520")
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
			decodeValue, err := reg.DecodeValue()
			if err != nil {
				t.Errorf("Error decoding value: %v", err)
			}
			t.Logf(
				`== Group[%v]
Index= %v
Address= %v
Tag= %v
BitMask= %v
Value = %v
DataType= %v
DataOrder= %v
DecodeValue.AsType= %v
DecodeValue.Float64= %v`,
				i, j, reg.Address, reg.Tag, reg.BitMask, reg.Value, reg.DataType, reg.DataOrder, decodeValue.AsType, decodeValue.Float64)
		}
	}
}

func Test_DeviceRegister_DecodeValue(t *testing.T) {
	tests := []struct {
		name      string
		register  DeviceRegister
		expect    DecodedValue
		expectErr bool
	}{
		{
			name: "bool",
			register: DeviceRegister{
				DataType: "bool",
				BitMask:  0x01,
				Value:    [8]byte{0x03, 0x00, 0x00, 0x00}, // 0x03 & 0x01 = 0x01
			},
			expect: DecodedValue{
				Raw:     []byte{0x03},
				Float64: 1.0,
				AsType:  uint8(0x01),
			},
			expectErr: false,
		},
		{
			name: "uint8",
			register: DeviceRegister{
				DataType: "uint8",
				Value:    [8]byte{0xFF, 0x00, 0x00, 0x00}, // 255
			},
			expect: DecodedValue{
				Raw:     []byte{0xFF},
				Float64: 255.0,
				AsType:  uint8(255),
			},
			expectErr: false,
		},
		{
			name: "int8",
			register: DeviceRegister{
				DataType: "int8",
				Value:    [8]byte{0x80, 0x00, 0x00, 0x00}, // -128
			},
			expect: DecodedValue{
				Raw:     []byte{0x80},
				Float64: -128.0,
				AsType:  int8(-128),
			},
			expectErr: false,
		},
		{
			name: "uint16",
			register: DeviceRegister{
				DataType: "uint16",
				Value:    [8]byte{0x12, 0x34, 0x00, 0x00}, // 0x1234 = 4660
			},
			expect: DecodedValue{
				Raw:     []byte{0x12, 0x34},
				Float64: 4660.0,
				AsType:  uint16(4660),
			},
			expectErr: false,
		},
		{
			name: "int16",
			register: DeviceRegister{
				DataType: "int16",
				Value:    [8]byte{0xFF, 0xFE, 0x00, 0x00}, // -2
			},
			expect: DecodedValue{
				Raw:     []byte{0xFF, 0xFE},
				Float64: -2.0,
				AsType:  int16(-2),
			},
			expectErr: false,
		},
		{
			name: "uint32",
			register: DeviceRegister{
				DataType: "uint32",
				Value:    [8]byte{0x12, 0x34, 0x56, 0x78}, // 0x12345678 = 305419896
			},
			expect: DecodedValue{
				Raw:     []byte{0x12, 0x34, 0x56, 0x78},
				Float64: 305419896.0,
				AsType:  uint32(305419896),
			},
			expectErr: false,
		},
		{
			name: "int32",
			register: DeviceRegister{
				DataType: "int32",
				Value:    [8]byte{0xFF, 0xFF, 0xFF, 0xFE}, // -2
			},
			expect: DecodedValue{
				Raw:     []byte{0xFF, 0xFF, 0xFF, 0xFE},
				Float64: -2.0,
				AsType:  int32(-2),
			},
			expectErr: false,
		},
		{
			name: "float32",
			register: DeviceRegister{
				DataType: "float32",
				Value:    [8]byte{0x42, 0x48, 0x00, 0x00}, // 50.0 in IEEE 754
			},
			expect: DecodedValue{
				Raw:     []byte{0x42, 0x48, 0x00, 0x00},
				Float64: 50.0,
				AsType:  float32(50.0),
			},
			expectErr: false,
		},
		{
			name: "float32-pi",
			register: DeviceRegister{
				DataType: "float32",
				Value:    [8]byte{0x40, 0x49, 0x0F, 0xDC}, // Pi
			},
			expect: DecodedValue{
				Raw:     []byte{0x40, 0x49, 0x0F, 0xDC},
				Float64: 3.1415929794311523,
				AsType:  float32(3.1415929794311523),
			},
			expectErr: false,
		},
		{
			name: "unsupported_data_type",
			register: DeviceRegister{
				DataType: "unsupported",
				Value:    [8]byte{0x00, 0x00, 0x00, 0x00},
			},
			expect:    DecodedValue{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.register.DecodeValue()
			if (err != nil) != tt.expectErr {
				t.Errorf("DecodeValue() error = %v, expectErr = %v", err, tt.expectErr)
				return
			}
			t.Log("== result.Raw, tt.expect.Raw == ", result.Raw, tt.expect.Raw)
			if !tt.expectErr {
				if result.Float64 != tt.expect.Float64 {
					t.Errorf("result.Float64 != tt.expect.Float64 = %v, want %v; result=%v expect=%v",
						result, tt.expect, result.Float64, tt.expect.Float64)
				}
				a := [8]byte{}
				b := [8]byte{}
				copy(a[:], result.Raw)
				copy(b[:], tt.expect.Raw)
				log.Printf("== a = %v, b = %v", a, b)
				if !compare2BytesEqual(a, b) {
					t.Errorf("XXX compareBytes() Raw = %v, want %v", result.Raw, tt.expect.Raw)
				}
				if hex.EncodeToString(a[:]) != hex.EncodeToString(b[:]) {
					t.Errorf("EncodeToString error Raw = %v, want %v", result.Raw, tt.expect.Raw)
				}
			}
		})
	}
}

// compare2BytesEqual parses two byte slices into unsigned integers and compares them
func compare2BytesEqual(a, b [8]byte) bool {
	// Parse both byte slices into unsigned integers
	var valA, valB uint64
	for i := 0; i < len(a); i++ {
		valA = (valA << 8) | uint64(a[i])
		valB = (valB << 8) | uint64(b[i])
	}

	// Compare the parsed values
	return valA == valB
}

// LoadRegisterFromCSV loads Modbus registers from a CSV file into a slice of DeviceRegister
func LoadRegisterFromCSV(filePath string) ([]DeviceRegister, error) {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Parse the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %v", err)
	}

	// Ensure there is at least a header row
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must contain at least a header and one data row")
	}

	// Parse the header row (optional, for validation)
	header := records[0]
	expectedHeader := []string{"Tag", "Alias", "Function", "SlaveId", "Address", "Frequency", "Quantity", "DataType", "BitMask", "DataOrder", "Weight"}
	if len(header) != len(expectedHeader) {
		return nil, fmt.Errorf("CSV header does not match expected format")
	}

	// Parse the data rows
	var registers []DeviceRegister
	for _, row := range records[1:] {
		if len(row) != len(expectedHeader) {
			return nil, fmt.Errorf("row length does not match header length: %v", row)
		}

		// Parse each field
		function, err := strconv.Atoi(row[2])
		if err != nil {
			return nil, fmt.Errorf("invalid Function value: %v", row[2])
		}

		slaveId, err := strconv.Atoi(row[3])
		if err != nil {
			return nil, fmt.Errorf("invalid SlaveId value: %v", row[3])
		}

		address, err := strconv.Atoi(row[4])
		if err != nil {
			return nil, fmt.Errorf("invalid Address value: %v", row[4])
		}

		frequency, err := strconv.ParseInt(row[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid Frequency value: %v", row[5])
		}

		quantity, err := strconv.Atoi(row[6])
		if err != nil {
			return nil, fmt.Errorf("invalid Quantity value: %v", row[6])
		}

		bitMask, err := strconv.Atoi(row[8])
		if err != nil {
			return nil, fmt.Errorf("invalid BitMask value: %v", row[8])
		}

		weight, err := strconv.ParseFloat(row[10], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid Weight value: %v", row[10])
		}

		// Create a DeviceRegister instance
		register := DeviceRegister{
			Tag:       row[0],
			Alias:     row[1],
			Function:  function,
			SlaverId:  byte(slaveId),
			Address:   uint16(address),
			Frequency: frequency,
			Quantity:  uint16(quantity),
			DataType:  row[7],
			BitMask:   uint16(bitMask),
			DataOrder: row[9],
			Weight:    weight,
		}

		// Append to the result slice
		registers = append(registers, register)
	}

	return registers, nil
}
func Test_LoadRegisterFromCSV(t *testing.T) {
	filePath := "./test/test-sheet.csv"
	registers, err := LoadRegisterFromCSV(filePath)
	if err != nil {
		t.Fatalf("Failed to load registers from CSV: %v", err)
	}
	// Print the loaded registers
	for _, register := range registers {
		t.Logf("Tag: %s, Alias: %s, Function: %d, SlaveId: %d, Address: %d, Frequency: %d, Quantity: %d, DataType: %s, BitMask: %d, DataOrder: %s, Weight: %.2f",
			register.Tag, register.Alias, register.Function, register.SlaverId, register.Address,
			register.Frequency, register.Quantity, register.DataType, register.BitMask, register.DataOrder,
			register.Weight)
	}
	grouped := GroupDeviceRegister(registers)
	jsonData, err := json.Marshal(grouped)
	if err != nil {
		t.Fatalf("error marshalling result: %v", err)
	}
	t.Logf("Grouped: %s", string(jsonData))
}
func Test_DeviceRegister_Decode_Bool_true_Value(t *testing.T) {
	tests := []struct {
		name      string
		register  DeviceRegister
		expect    DecodedValue
		expectErr bool
	}{}
	for i := 0; i < 16; i++ {
		tests = append(tests, struct {
			name      string
			register  DeviceRegister
			expect    DecodedValue
			expectErr bool
		}{
			name: fmt.Sprintf("bool-%v", i),
			register: DeviceRegister{
				DataType: "bool",
				BitMask:  uint16(i),
				Value:    [8]byte{0xFF, 0xFF, 0xFF, 0xFF},
			},
			expect: DecodedValue{
				Raw:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
				Float64: 1.0,
				AsType:  bool(true),
			},
			expectErr: false,
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.register.DecodeValue()
			if (err != nil) != tt.expectErr {
				t.Errorf("DecodeValue() error = %v, expectErr = %v", err, tt.expectErr)
				return
			}
			t.Log("== ", result.String())
			if !tt.expectErr {
				if result.Float64 != tt.expect.Float64 {
					t.Errorf("result.Float64!= tt.expect.Float64 = %v, want %v; result=%v expect=%v",
						result, tt.expect, result.Float64, tt.expect.Float64)
				}
			}

		})
	}
}
func Test_DeviceRegister_Decode_Bool_false_Value(t *testing.T) {
	tests := []struct {
		name      string
		register  DeviceRegister
		expect    DecodedValue
		expectErr bool
	}{}
	for i := 0; i < 16; i++ {
		tests = append(tests, struct {
			name      string
			register  DeviceRegister
			expect    DecodedValue
			expectErr bool
		}{
			name: fmt.Sprintf("bool-%v", i),
			register: DeviceRegister{
				DataType: "bool",
				BitMask:  uint16(i),
				Value:    [8]byte{0x0},
			},
			expect: DecodedValue{
				Raw:     []byte{0x0},
				Float64: 0,
				AsType:  bool(true),
			},
			expectErr: false,
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.register.DecodeValue()
			if (err != nil) != tt.expectErr {
				t.Errorf("DecodeValue() error = %v, expectErr = %v", err, tt.expectErr)
				return
			}
			t.Log("== ", result.String())
			if !tt.expectErr {
				if result.Float64 != tt.expect.Float64 {
					t.Errorf("result.Float64!= tt.expect.Float64 = %v, want %v; result=%v expect=%v",
						result, tt.expect, result.Float64, tt.expect.Float64)
				}
			}

		})
	}
}
