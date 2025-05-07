package modbus

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

// TestReorderBytes verifies byte order handling
func TestReorderBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	tests := []struct {
		name     string
		order    string
		expected []byte
	}{
		{"Single byte A", "A", []byte{0x01}},
		{"Little-endian AB", "AB", []byte{0x01, 0x02}},
		{"Big-endian BA", "BA", []byte{0x02, 0x01}},
		{"Little-endian 4-byte ABCD", "ABCD", []byte{0x01, 0x02, 0x03, 0x04}},
		{"Big-endian 4-byte DCBA", "DCBA", []byte{0x04, 0x03, 0x02, 0x01}},
		{"Word-swapped BADC", "BADC", []byte{0x02, 0x01, 0x04, 0x03}},
		{"Word-swapped CDAB", "CDAB", []byte{0x03, 0x04, 0x01, 0x02}},
		{"Little-endian 8-byte ABCDEFGH", "ABCDEFGH", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
		{"Big-endian 8-byte HGFEDCBA", "HGFEDCBA", []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}},
		{"Word-swapped 8-byte BADCFEHG", "BADCFEHG", []byte{0x02, 0x01, 0x04, 0x03, 0x06, 0x05, 0x08, 0x07}},
		{"Word-swapped 8-byte GHEFCDAB", "GHEFCDAB", []byte{0x07, 0x08, 0x05, 0x06, 0x03, 0x04, 0x01, 0x02}},
		{"Unknown format", "XYZ", data},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reorderBytes(data, tt.order)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("reorderBytes(%v, %s) = %v, want %v", data, tt.order, result, tt.expected)
			}
		})
	}

	// Test with insufficient data length
	shortData := []byte{0x01, 0x02}
	result := reorderBytes(shortData, "ABCD")
	if len(result) != len(shortData) {
		t.Errorf("reorderBytes should return original data when length is insufficient, got %v", result)
	}
}

// TestDecodeValue_Int32 tests decoding of int32 values
func TestDecodeValue_Int32(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType int32
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Int32 positive value",
			value:      []byte{0x5A, 0xA5, 0xA5, 0x5A},
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: 0x5AA5A55A,
			expectF64:  1520805210.0,
		},
		{
			name:       "Int32 negative value",
			value:      []byte{0x80, 0x00, 0x00, 0x00},
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: -2147483648,
			expectF64:  -2147483648.0,
		},
		{
			name:       "Int32 big-endian (DCBA)",
			value:      []byte{0x00, 0x00, 0x00, 0x80},
			dataOrder:  "DCBA",
			weight:     1.0,
			expectType: -2147483648,
			expectF64:  -2147483648.0,
		},
		{
			name:       "Int32 word-swapped BADC",
			value:      []byte{0x00, 0x80, 0x00, 0x00},
			dataOrder:  "BADC",
			weight:     1.0,
			expectType: -2147483648,
			expectF64:  -2147483648.0,
		},
		{
			name:       "Int32 with weight -2.5",
			value:      []byte{0x00, 0x00, 0x00, 0x01},
			dataOrder:  "ABCD",
			weight:     -2.5,
			expectType: 1,
			expectF64:  -2.5, // 1 * -2.5 = -2.5
		},
		{
			name:      "Int32 insufficient bytes",
			value:     []byte{0x5A, 0xA5, 0xA5},
			dataOrder: "ABCD",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "int32",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(int32); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be int32, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_Float32 tests decoding of float32 values
func TestDecodeValue_Float32(t *testing.T) {
	// Helper function to create IEEE-754 float32 bytes
	createFloat32Bytes := func(f float32) []byte {
		bits := math.Float32bits(f)
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, bits)
		return bytes
	}

	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType float32
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Float32 positive value 123.456",
			value:      createFloat32Bytes(123.456),
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: 123.456,
			expectF64:  123.456,
		},
		{
			name:       "Float32 negative value -123.456",
			value:      createFloat32Bytes(-123.456),
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: -123.456,
			expectF64:  -123.456,
		},
		{
			name: "Float32 big-endian (DCBA)",
			value: []byte{
				createFloat32Bytes(123.456)[3],
				createFloat32Bytes(123.456)[2],
				createFloat32Bytes(123.456)[1],
				createFloat32Bytes(123.456)[0],
			},
			dataOrder:  "DCBA",
			weight:     1.0,
			expectType: 123.456,
			expectF64:  123.456,
		},
		{
			name: "Float32 word-swapped BADC",
			value: []byte{
				createFloat32Bytes(123.456)[1],
				createFloat32Bytes(123.456)[0],
				createFloat32Bytes(123.456)[3],
				createFloat32Bytes(123.456)[2],
			},
			dataOrder:  "BADC",
			weight:     1.0,
			expectType: 123.456,
			expectF64:  123.456,
		},
		{
			name:       "Float32 with weight 2.5",
			value:      createFloat32Bytes(123.456),
			dataOrder:  "ABCD",
			weight:     2.5,
			expectType: 123.456,
			expectF64:  308.64, // 123.456 * 2.5 = 308.64
		},
		{
			name:      "Float32 insufficient bytes",
			value:     []byte{0x5A, 0xA5, 0xA5},
			dataOrder: "ABCD",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "float32",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(float32); ok {
				// Use fuzzy comparison for floating point
				if math.Abs(float64(val-tt.expectType)) > 0.0001 {
					t.Errorf("Expected type value %f, got %f", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be float32, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}
func TestDecodeValue_Float64(t *testing.T) {
	// Helper function to create IEEE-754 float64 bytes
	createFloat64Bytes := func(f float64) []byte {
		bits := math.Float64bits(f)
		bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, bits)
		return bytes
	}

	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType float64
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Float64 positive value 123.456",
			value:      createFloat64Bytes(123.456),
			dataOrder:  "ABCDEFGH",
			weight:     1.0,
			expectType: 123.456,
			expectF64:  123.456,
		},
		{
			name:       "Float64 negative value -123.456",
			value:      createFloat64Bytes(-123.456),
			dataOrder:  "ABCDEFGH",
			weight:     1.0,
			expectType: -123.456,
			expectF64:  -123.456,
		},
		{
			name: "Float64 big-endian (HGFEDCBA)",
			value: []byte{
				createFloat64Bytes(123.456)[7],
				createFloat64Bytes(123.456)[6],
				createFloat64Bytes(123.456)[5],
				createFloat64Bytes(123.456)[4],
				createFloat64Bytes(123.456)[3],
				createFloat64Bytes(123.456)[2],
				createFloat64Bytes(123.456)[1],
				createFloat64Bytes(123.456)[0],
			},
			dataOrder:  "HGFEDCBA",
			weight:     1.0,
			expectType: 123.456,
			expectF64:  123.456,
		},
		{
			name:       "Float64 with weight 2.5",
			value:      createFloat64Bytes(123.456),
			dataOrder:  "ABCDEFGH",
			weight:     2.5,
			expectType: 123.456,
			expectF64:  308.64, // 123.456 * 2.5 = 308.64
		},
		{
			name:      "Float64 insufficient bytes",
			value:     []byte{0x5A, 0xA5, 0xA5, 0x5A, 0x5A, 0xA5, 0xA5},
			dataOrder: "ABCDEFGH",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "float64",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(float64); ok {
				if math.Abs(val-tt.expectType) > 0.0001 {
					t.Errorf("Expected type value %f, got %f", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be float64, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_UInt16 tests decoding of uint16 values
func TestDecodeValue_UInt16(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType uint16
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "UInt16 value 0x5AA5",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5AA5,
			expectF64:  23205.0,
		},
		{
			name:       "UInt16 byte-swapped 0x5AA5",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "BA",
			weight:     1.0,
			expectType: 0xA55A,
			expectF64:  42330.0,
		},
		{
			name:       "UInt16 with weight 2.5",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "AB",
			weight:     2.5,
			expectType: 0x5AA5,
			expectF64:  58012.5, // 23205 * 2.5 = 58012.5
		},
		{
			name:      "UInt16 insufficient bytes",
			value:     []byte{0x5A},
			dataOrder: "AB",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "uint16",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(uint16); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be uint16, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_Int16 tests decoding of int16 values
func TestDecodeValue_Int16(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType int16
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Int16 positive value 0x5AA5",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5AA5,
			expectF64:  23205.0,
		},
		{
			name:       "Int16 negative value 0x8000",
			value:      []byte{0x80, 0x00},
			dataOrder:  "AB",
			weight:     1.0,
			expectType: -32768,
			expectF64:  -32768.0,
		},
		{
			name:       "Int16 byte-swapped negative",
			value:      []byte{0x00, 0x80},
			dataOrder:  "BA",
			weight:     1.0,
			expectType: -32768,
			expectF64:  -32768.0,
		},
		{
			name:       "Int16 with weight 2.5",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "AB",
			weight:     2.5,
			expectType: 0x5AA5,
			expectF64:  58012.5, // 23205 * 2.5 = 58012.5
		},
		{
			name:      "Int16 insufficient bytes",
			value:     []byte{0x5A},
			dataOrder: "AB",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "int16",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(int16); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be int16, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_UInt32 tests decoding of uint32 values
func TestDecodeValue_UInt32(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType uint32
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "UInt32 positive value 0x5AA5A55A",
			value:      []byte{0x5A, 0xA5, 0xA5, 0x5A},
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: 0x5AA5A55A,
			expectF64:  1520805210.0,
		},
		{
			name:       "UInt32 negative value 0x80000000",
			value:      []byte{0x80, 0x00, 0x00, 0x00},
			dataOrder:  "ABCD",
			weight:     1.0,
			expectType: 0x80000000,
			expectF64:  2147483648.0,
		},
		{
			name:       "UInt32 byte-swapped 0x5AA5A55A",
			value:      []byte{0x5A, 0xA5, 0xA5, 0x5A},
			dataOrder:  "DCBA",
			weight:     1.0,
			expectType: 0x5AA5A55A,
			expectF64:  1520805210.0,
		},
		{
			name:       "UInt32 with weight 2.5",
			value:      []byte{0x5A, 0xA5, 0xA5, 0x5A},
			dataOrder:  "ABCD",
			weight:     2.5,
			expectType: 0x5AA5A55A,
			expectF64:  3802013025.0, // 1520805210 * 2.5 = 3802013025.0
		},
		{
			name:      "UInt32 insufficient bytes",
			value:     []byte{0x5A, 0xA5, 0xA5},
			dataOrder: "ABCD",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "uint32",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(uint32); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be uint32, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestCheckBit verifies bit checking functionality
func TestCheckBit(t *testing.T) {
	// Test value 0x5A (01011010)
	value := uint16(0x5A)

	// Expected bit values for 0x5A
	expected := []bool{false, true, false, true, true, false, true, false}

	for i := 0; i < 8; i++ {
		result := CheckBit(value, uint16(i))
		if result != expected[i] {
			t.Errorf("CheckBit(0x5A, %d) = %v, want %v", i, result, expected[i])
		}
	}

	// Test out of range index
	if CheckBit(value, 16) {
		t.Errorf("CheckBit(0x5A, 16) = true, want false (out of range)")
	}

	// Test all 16 bits with alternating pattern
	value = 0xAAAA // 1010 1010 1010 1010
	for i := 0; i < 16; i++ {
		expected := i%2 == 1 // Odd indices should be true
		result := CheckBit(value, uint16(i))
		if result != expected {
			t.Errorf("CheckBit(0xAAAA, %d) = %v, want %v", i, result, expected)
		}
	}
}

// TestDecodeValue_Bitfield tests decoding of bitfield values
func TestDecodeValue_Bitfield(t *testing.T) {
	// Test cases for bitfield
	tests := []struct {
		name       string
		value      []byte
		bitMask    uint16
		dataOrder  string
		weight     float64
		expectType uint16
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Bitfield value 0x5AA5",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0xFFFF,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5AA5,
			expectF64:  23205.0,
		},
		{
			name:       "Bitfield value with weight 2.5",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0xFFFF,
			dataOrder:  "AB",
			weight:     2.5,
			expectType: 0x5AA5,
			expectF64:  58012.5, // 23205 * 2.5 = 58012.5
		},
		{
			name:      "Bitfield insufficient bytes",
			value:     []byte{0x5A},
			bitMask:   0xFFFF,
			dataOrder: "AB",
			expectErr: true,
		},
		{
			name:       "Bitfield with bitmask 0x00FF",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0x00FF,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x00A5,
			expectF64:  165.0,
		},
		{
			name:       "Bitfield with bitmask 0xFF00",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0xFF00,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5A00,
			expectF64:  23040.0,
		},
		{
			name:       "Bitfield with bitmask 0x0000",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0x0000,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x0000,
			expectF64:  0.0,
		},
		{
			name:       "Bitfield with bitmask 0xFFFF",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0xFFFF,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5AA5,
			expectF64:  23205.0,
		},
		{
			name:       "Bitfield with bitmask 0x7FFF",
			value:      []byte{0x5A, 0xA5},
			bitMask:    0x7FFF,
			dataOrder:  "AB",
			weight:     1.0,
			expectType: 0x5AA5,
			expectF64:  23205.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "bitfield",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
				BitMask:   tt.bitMask,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(uint16); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be uint16, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_Bool tests decoding of boolean values
func TestDecodeValue_Bool(t *testing.T) {
	// Test cases for boolean
	tests := []struct {
		name        string
		value       []byte
		bitPosition uint16
		dataOrder   string
		expectType  bool
		expectF64   float64
		expectErr   bool
	}{
		{
			name:        "Bool bit 0 set",
			value:       []byte{0x01, 0x00},
			bitPosition: 8,
			dataOrder:   "AB",
			expectType:  true,
			expectF64:   1.0,
		},
		{
			name:        "Bool bit 0 clear",
			value:       []byte{0x02, 0x00},
			bitPosition: 0,
			dataOrder:   "AB",
			expectType:  false,
			expectF64:   0.0,
		},
		{
			name:        "Bool bit 7 set",
			value:       []byte{0x80, 0x00},
			bitPosition: 7,
			dataOrder:   "BA",
			expectType:  true,
			expectF64:   1.0,
		},
		{
			name:        "Bool bit 15 set",
			value:       []byte{0xFF, 0x00},
			bitPosition: 15,
			dataOrder:   "AB",
			expectType:  true,
			expectF64:   1.0,
		},
		{
			name:        "Bool big-endian with bit 7",
			value:       []byte{0x00, 0b_10000000},
			bitPosition: 7,
			dataOrder:   "AB",
			expectType:  true,
			expectF64:   1.0,
		},
		{
			name:        "Bool insufficient bytes",
			value:       []byte{0x01},
			bitPosition: 0,
			dataOrder:   "AB",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:    "bool",
				DataOrder:   tt.dataOrder,
				BitPosition: tt.bitPosition,
				Value:       tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(bool); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %v, got %v", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be bool, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_Byte tests decoding of byte values
func TestDecodeValue_Byte(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType byte
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Byte value 0x5A",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     1.0,
			expectType: 0x5A,
			expectF64:  90.0,
		},
		{
			name:       "Byte value with weight 2.5",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     2.5,
			expectType: 0x5A,
			expectF64:  90.0, // 90 * 2.5 = 225
		},
		{
			name:       "Byte with multiple bytes (takes first)",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "A",
			weight:     1.0,
			expectType: 0x5A,
			expectF64:  90.0,
		},
		{
			name:      "Byte empty value",
			value:     []byte{},
			dataOrder: "A",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "byte",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(byte); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be byte, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_UInt8 tests decoding of uint8 values
func TestDecodeValue_UInt8(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType uint8
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "UInt8 value 0x5A",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     1.0,
			expectType: 0x5A,
			expectF64:  90.0,
		},
		{
			name:       "UInt8 value with weight 2.5",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     2.5,
			expectType: 0x5A,
			expectF64:  225.0, // 90 * 2.5 = 225
		},
		{
			name:       "UInt8 with multiple bytes (takes first)",
			value:      []byte{0x5A, 0xA5},
			dataOrder:  "A",
			weight:     1.0,
			expectType: 0x5A,
			expectF64:  90.0,
		},
		{
			name:      "UInt8 empty value",
			value:     []byte{},
			dataOrder: "A",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "uint8",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(uint8); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be uint8, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

// TestDecodeValue_Int8 tests decoding of int8 values
func TestDecodeValue_Int8(t *testing.T) {
	tests := []struct {
		name       string
		value      []byte
		dataOrder  string
		weight     float64
		expectType int8
		expectF64  float64
		expectErr  bool
	}{
		{
			name:       "Int8 positive value 0x5A",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     1.0,
			expectType: 0x5A,
			expectF64:  90.0,
		},
		{
			name:       "Int8 negative value 0x80",
			value:      []byte{0x80},
			dataOrder:  "A",
			weight:     1.0,
			expectType: -128,
			expectF64:  -128.0,
		},
		{
			name:       "Int8 with weight 2.5",
			value:      []byte{0x5A},
			dataOrder:  "A",
			weight:     2.5,
			expectType: 0x5A,
			expectF64:  225.0, // 90 * 2.5 = 225
		},
		{
			name:      "Int8 empty value",
			value:     []byte{},
			dataOrder: "A",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{
				DataType:  "int8",
				DataOrder: tt.dataOrder,
				Weight:    tt.weight,
				Value:     tt.value,
			}

			result, err := reg.DecodeValue()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if val, ok := result.AsType.(int8); ok {
				if val != tt.expectType {
					t.Errorf("Expected type value %d, got %d", tt.expectType, val)
				}
			} else {
				t.Errorf("Expected AsType to be int8, got %T", result.AsType)
			}

			if !FuzzyEqual(result.Float64, tt.expectF64) {
				t.Errorf("Expected float64 value %f, got %f", tt.expectF64, result.Float64)
			}
		})
	}
}

func Test_GroupDeviceRegisterWithLogicalContinuity(t *testing.T) {
	tests := []struct {
		name      string
		registers []DeviceRegister
		expected  [][]DeviceRegister
	}{
		{
			name: "Single group with logical continuity1",
			registers: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 5, ReadQuantity: 2},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
					{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
					{SlaverId: 1, ReadAddress: 5, ReadQuantity: 2},
				},
			},
		},
		{
			name: "Single group with logical continuity2",
			registers: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 0, ReadQuantity: 4},
				{SlaverId: 1, ReadAddress: 4, ReadQuantity: 4},
				{SlaverId: 1, ReadAddress: 8, ReadQuantity: 4},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 0, ReadQuantity: 4},
					{SlaverId: 1, ReadAddress: 4, ReadQuantity: 4},
					{SlaverId: 1, ReadAddress: 8, ReadQuantity: 4},
				},
			},
		},
		{
			name: "Multiple groups with logical continuity",
			registers: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 7, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 9, ReadQuantity: 2},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
					{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
				},
				{
					{SlaverId: 1, ReadAddress: 7, ReadQuantity: 2},
					{SlaverId: 1, ReadAddress: 9, ReadQuantity: 2},
				},
			},
		},
		{
			name: "Registers with different SlaverIds",
			registers: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
				{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
				{SlaverId: 2, ReadAddress: 1, ReadQuantity: 2},
				{SlaverId: 2, ReadAddress: 3, ReadQuantity: 2},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 1, ReadQuantity: 2},
					{SlaverId: 1, ReadAddress: 3, ReadQuantity: 2},
				},
				{
					{SlaverId: 2, ReadAddress: 1, ReadQuantity: 2},
					{SlaverId: 2, ReadAddress: 3, ReadQuantity: 2},
				},
			},
		},
		{
			name:      "No registers",
			registers: []DeviceRegister{},
			expected:  [][]DeviceRegister{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupDeviceRegisterWithLogicalContinuity(tt.registers)
			beautifulPrint(t, result)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GroupDeviceRegisterWithLogicalContinuity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func beautifulPrint(t *testing.T, r [][]DeviceRegister) {
	for _, group := range r {
		for _, reg := range group {
			t.Logf("SlaverId: %d, ReadAddress: %d, ReadQuantity: %d", reg.SlaverId, reg.ReadAddress, reg.ReadQuantity)
		}
	}
}
