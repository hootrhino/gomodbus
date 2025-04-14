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
	"time"
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
		{Tag: "F", Alias: "A6", SlaverId: 1, Function: 3, ReadAddress: 1, ReadQuantity: 1},
		{Tag: "A", Alias: "A1", SlaverId: 1, Function: 3, ReadAddress: 2, ReadQuantity: 1},
		{Tag: "B", Alias: "A2", SlaverId: 1, Function: 3, ReadAddress: 4, ReadQuantity: 1},
		{Tag: "C", Alias: "A3", SlaverId: 1, Function: 3, ReadAddress: 5, ReadQuantity: 1},
		{Tag: "D", Alias: "A4", SlaverId: 1, Function: 3, ReadAddress: 8, ReadQuantity: 1},
		{Tag: "E", Alias: "A5", SlaverId: 1, Function: 3, ReadAddress: 9, ReadQuantity: 1},
		{Tag: "G", Alias: "A7", SlaverId: 1, Function: 3, ReadAddress: 10, ReadQuantity: 1},
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
	defer client.Close()
	result := client.ReadGroupedRegisterValue(input)
	for i, group := range result {
		for j, reg := range group {
			t.Logf("======= group->%v  reg=%v  Address= %v  Tag= %v", i, j, reg.ReadAddress, reg.Tag)
		}
	}
}

// go test -timeout 30s -run ^Test_GroupDevice_UART_125_Registers$ github.com/hootrhino/gomodbus -v -count=1
func Test_GroupDevice_UART_125_Registers(t *testing.T) {
	// Group 1: 1-25
	input1 := make([]DeviceRegister, 10)
	for i := 0; i < 10; i++ {
		input1[i].ReadAddress = uint16(i + 1)
		input1[i].Tag = fmt.Sprintf("Tag%d", i+1)
		input1[i].Alias = fmt.Sprintf("Alias%d", i+1)
		input1[i].Function = 3
		input1[i].SlaverId = 1
		input1[i].Frequency = 1
		input1[i].ReadQuantity = 1
		input1[i].DataType = "uint16"
		input1[i].DataOrder = "ABCD"
		input1[i].Weight = 1.0
		input1[i].Value = [8]byte{0, 0, 0, 0}
	}
	input2 := make([]DeviceRegister, 10)
	for i := 26; i < 36; i++ {
		input2[i-26].ReadAddress = uint16(i + 1)
		input2[i-26].Tag = fmt.Sprintf("Tag%d", i+1)
		input2[i-26].Alias = fmt.Sprintf("Alias%d", i+1)
		input2[i-26].Function = 3
		input2[i-26].SlaverId = 1
		input2[i-26].Frequency = 1
		input2[i-26].ReadQuantity = 1
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
	defer client.Close()
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
	defer client.Close()
	input1 := make([]DeviceRegister, 1)
	for i := 0; i < 16; i++ {
		t.Log("======= Test_Group_UART_Device_1_Bool_Register", i)
		input1[0].ReadAddress = uint16(4) // 0000 0000 0000 0100
		input1[0].Tag = "Tag1"
		input1[0].Alias = "Alias1"
		input1[0].Function = 3
		input1[0].SlaverId = 1
		input1[0].Frequency = 1
		input1[0].ReadQuantity = 1
		input1[0].DataType = "bool"
		input1[0].DataOrder = "A"
		input1[0].Weight = 1.0
		input1[0].Value = [8]byte{0, 0, 0, 0}
		input1[0].BitPosition = uint16(i)
		testGroup(t, client, input1)
	}

}

// go test -timeout 30s -run ^Test_Group_TCP_Device_125_Registers$ github.com/hootrhino/gomodbus -v -count=1
func Test_Group_TCP_Device_125_Registers(t *testing.T) {
	// Group 1: 1-25
	input1 := make([]DeviceRegister, 10)
	for i := 0; i < 10; i++ {
		input1[i].ReadAddress = uint16(i + 1)
		input1[i].Tag = fmt.Sprintf("Tag%d", i+1)
		input1[i].Alias = fmt.Sprintf("Alias%d", i+1)
		input1[i].Function = 3
		input1[i].SlaverId = 1
		input1[i].Frequency = 1
		input1[i].ReadQuantity = 1
		input1[i].DataType = "uint16"
		input1[i].DataOrder = "ABCD"
		input1[i].Weight = 1.0
		input1[i].Value = [8]byte{0, 0, 0, 0}
	}
	input2 := make([]DeviceRegister, 10)
	for i := 26; i < 36; i++ {
		input2[i-26].ReadAddress = uint16(i + 1)
		input2[i-26].Tag = fmt.Sprintf("Tag%d", i+1)
		input2[i-26].Alias = fmt.Sprintf("Alias%d", i+1)
		input2[i-26].Function = 3
		input2[i-26].SlaverId = 1
		input2[i-26].Frequency = 1
		input2[i-26].ReadQuantity = 1
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
	defer client.Close()
	testGroup(t, client, input1)
	testGroup(t, client, input2)
}

// go test -timeout 30s -run ^Test_Group_TCP_Device_1_Bool_Register$ github.com/hootrhino/gomodbus -v -count=1
func testGroup(t *testing.T, client Client, input []DeviceRegister) {
	result := client.ReadGroupedRegisterValue(input)
	for i, group := range result {
		t.Logf("== ReadGroupedRegisterValue.[%v]", i)
		for _, reg := range group {
			decodeValue, err := reg.DecodeValue()
			if err != nil {
				t.Errorf("Error decoding value: %v", err)
			}
			t.Logf(
				`
============= Value =============
R.Tag: %v
R.Alias: %v
R.SlaverId: %v
R.Function: %v
R.Address: %v
R.Quantity: %v
R.DataOrder: %v
R.DataType: %v
R.Weight: %v
R.Value: %v
R.Status: %v
---------------------------------
V.AsType: %v
V.Float64: %v
=================================
`,
				reg.Tag, reg.Alias, reg.SlaverId,
				reg.Function, reg.ReadAddress, reg.ReadQuantity,
				reg.DataOrder, reg.DataType, reg.Weight, reg.Value, reg.Status,
				decodeValue.AsType, decodeValue.GetFloat64Value(4))
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
				DataType:    "bool",
				BitPosition: 0x01,
				Value:       [8]byte{0x03, 0x00, 0x00, 0x00}, // 0x03 & 0x01 = 0x01
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

func LoadRegisterFromCSV(filePath string) ([]DeviceRegister, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// 跳过表头
	if len(records) == 0 {
		return nil, nil
	}
	records = records[1:]

	var registers []DeviceRegister
	for _, record := range records {
		if len(record) != 12 {
			return nil, fmt.Errorf("invalid record length: %d", len(record))
		}

		slaverId, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, err
		}

		function, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, err
		}

		readAddress, err := strconv.Atoi(record[4])
		if err != nil {
			return nil, err
		}

		readQuantity, err := strconv.Atoi(record[5])
		if err != nil {
			return nil, err
		}

		bitPosition, err := strconv.Atoi(record[8])
		if err != nil {
			return nil, err
		}

		bitMask, err := strconv.Atoi(record[9])
		if err != nil {
			return nil, err
		}

		weight, err := strconv.ParseFloat(record[10], 64)
		if err != nil {
			return nil, err
		}

		frequency, err := strconv.ParseUint(record[11], 10, 64)
		if err != nil {
			return nil, err
		}

		register := DeviceRegister{
			Tag:          record[0],
			Alias:        record[1],
			SlaverId:     uint8(slaverId),
			Function:     uint8(function),
			ReadAddress:  uint16(readAddress),
			ReadQuantity: uint16(readQuantity),
			DataType:     record[6],
			DataOrder:    record[7],
			BitPosition:  uint16(bitPosition),
			BitMask:      uint16(bitMask),
			Weight:       weight,
			Frequency:    frequency,
		}
		registers = append(registers, register)
	}

	return registers, nil
}

// go test -timeout 30s -run ^Test_LoadRegisterFromCSV$ github.com/hootrhino/gomodbus -v -count=1
type testMqttData struct {
	Header map[string]any `json:"header"`
	Body   map[string]any `json:"body"`
}

func Test_LoadRegisterFromCSV(t *testing.T) {
	filePath := "./test/modbus_registers.csv"
	registers, err1 := LoadRegisterFromCSV(filePath)
	if err1 != nil {
		t.Fatalf("Failed to load registers from CSV: %v", err1)
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
	defer client.Close()
	manager := NewRegisterManager(client, 10)
	manager.LoadRegisters(registers)
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Log(err)
			}
			testData := testMqttData{
				Header: map[string]any{
					"tag":   register.Tag,
					"alias": register.Alias,
				},
				Body: map[string]any{
					"value": value.AsType,
				},
			}
			jsonData, err := json.Marshal(testData)
			if err != nil {
				t.Error(err)
			}
			t.Log(string(jsonData))

			// t.Log("== ", register.Tag, register.Alias, register.DataType,
			// 	register.DataOrder, register.BitPosition, register.Weight, register.Frequency, value)
		}
	})
	manager.Start()
	for range 10 {
		manager.ReadGroupedData()
	}

	time.Sleep(4 * time.Second)
	manager.Stop()
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
				DataType:    "bool",
				BitPosition: uint16(i),
				Value:       [8]byte{0xFF, 0xFF, 0xFF, 0xFF},
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
				DataType:    "bool",
				BitPosition: uint16(i),
				Value:       [8]byte{0x0},
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

// go test -benchmem -run=^$ -bench ^Benchmark_Decode_10_TCP_Registers$ github.com/hootrhino/gomodbus -v -count=10000 -benchtime=10s
func Benchmark_Decode_10_TCP_Registers(b *testing.B) {
	b.ResetTimer()

	input1 := []DeviceRegister{}
	{
		// Func1
		for i := range 16 {
			reg1 := DeviceRegister{}
			reg1.Tag = fmt.Sprintf("Tag-bool-%d-%d", 1, i)
			reg1.Alias = fmt.Sprintf("Alias-bool-%d-%d", 1, i)
			reg1.SlaverId = 1
			reg1.Function = 1
			reg1.ReadAddress = 0
			reg1.ReadQuantity = 1
			reg1.DataType = "bool"
			reg1.DataOrder = "A"
			reg1.Frequency = 10
			reg1.Weight = 1
			reg1.BitPosition = uint16(i)
			reg1.Value = [8]byte{0}
			input1 = append(input1, reg1)
		}
	}
	{
		// Func2
		for i := 10; i < 20; i++ {
			reg1 := DeviceRegister{}
			reg1.Tag = fmt.Sprintf("Tag-%d-%d", 2, i)
			reg1.Alias = fmt.Sprintf("Alias-%d-%d", 2, i)
			reg1.SlaverId = 1
			reg1.Function = 2
			reg1.ReadAddress = uint16(i)
			reg1.ReadQuantity = 1
			reg1.DataType = "uint16"
			reg1.DataOrder = "AB"
			reg1.Frequency = 10
			reg1.Weight = 1
			reg1.Value = [8]byte{0}
			input1 = append(input1, reg1)
		}
	}
	{
		// Func3
		for i := 20; i < 30; i++ {
			reg1 := DeviceRegister{}
			reg1.Tag = fmt.Sprintf("Tag-%d-%d", 3, i)
			reg1.Alias = fmt.Sprintf("Alias-%d-%d", 3, i)
			reg1.SlaverId = 1
			reg1.Function = 3
			reg1.ReadAddress = uint16(i)
			reg1.ReadQuantity = 1
			reg1.DataType = "uint16"
			reg1.DataOrder = "AB"
			reg1.Frequency = 10
			reg1.Weight = 1
			reg1.Value = [8]byte{0}
			input1 = append(input1, reg1)
		}
	}
	{
		// Func4
		for i := 30; i < 40; i++ {
			reg1 := DeviceRegister{}
			reg1.Tag = fmt.Sprintf("Tag-%d-%d", 4, i)
			reg1.Alias = fmt.Sprintf("Alias-%d-%d", 4, i)
			reg1.SlaverId = 1
			reg1.Function = 4
			reg1.ReadAddress = uint16(i)
			reg1.ReadQuantity = 1
			reg1.DataType = "uint16"
			reg1.DataOrder = "AB"
			reg1.Frequency = 10
			reg1.Weight = 1
			reg1.Value = [8]byte{0}
			input1 = append(input1, reg1)
		}
	}
	// Group the registers
	handler := NewRTUClientHandler("COM17")
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)

	err := handler.Connect()
	if err != nil {
		b.Fatal(err)
	}
	defer handler.Close()
	client := NewClient(handler)
	defer client.Close()
	acc := 1000
	b.Run("Decode_10_TCP_Registers", func(b *testing.B) {
		if acc > 1000 {
			b.StopTimer()
		}
		acc--
		result := client.ReadGroupedRegisterValue(input1)
		for _, group := range result {
			for _, reg := range group {
				b.Log("== ", reg.String())
			}
		}

	})

}
func Test_Decode_Print_Registers(t *testing.T) {
	registers := []DeviceRegister{
		{SlaverId: 1, ReadAddress: 1, ReadQuantity: 1, Tag: "tag1"},
		{SlaverId: 1, ReadAddress: 1, ReadQuantity: 1, Tag: "tag1_duplicate"}, // Duplicate
		{SlaverId: 1, ReadAddress: 2, ReadQuantity: 1, Tag: "tag2"},
		{SlaverId: 2, ReadAddress: 1, ReadQuantity: 1, Tag: "tag3"},
		{SlaverId: 2, ReadAddress: 10, ReadQuantity: 1, Tag: "tag4"},
		{SlaverId: 2, ReadAddress: 10, ReadQuantity: 1, Tag: "tag4_duplicate"}, // Duplicate
		{SlaverId: 3, ReadAddress: 0, ReadQuantity: 1, Tag: "tag5"},
		{SlaverId: 3, ReadAddress: 1, ReadQuantity: 1, Tag: "tag6"},
	}

	groups := GroupDeviceRegister(registers)
	PrintGroups(groups)
}

// Simple function to print groups for debugging/demonstration
func PrintGroups(groups [][]DeviceRegister) {
	for i, group := range groups {
		if len(group) == 0 {
			continue
		}
		fmt.Printf("Group %d (SlaveId=%d):\n", i+1, group[0].SlaverId)
		fmt.Printf("  Registers: ")
		for _, reg := range group {
			fmt.Printf("(Addr=%d Qty=%d Tag=%s) ", reg.ReadAddress, reg.ReadQuantity, reg.Tag)
		}
		fmt.Printf("\n")
	}
}
