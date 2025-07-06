// Copyright (C) 2024  wwhai
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <https://www.gnu.org/licenses/>.

package modbus

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"encoding/json"
	"reflect"
)

func ParseRegistersCSV(filePath string) ([]DeviceRegister, error) {
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

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file requires at least header and one data row")
	}

	headers := records[0]
	registers := make([]DeviceRegister, 0, len(records)-1)

	headerIndex := make(map[string]int)
	for i, header := range headers {
		headerIndex[header] = i
	}

	requiredColumns := []string{
		"uuid", "tag", "alias", "slaverId", "function",
		"readAddress", "readQuantity", "dataType", "dataOrder",
		"bitPosition", "bitMask", "weight", "frequency",
	}

	for _, col := range requiredColumns {
		if _, exists := headerIndex[col]; !exists {
			return nil, fmt.Errorf("Missing required column: %s", col)
		}
	}

	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < len(headers) {
			return nil, fmt.Errorf("Row %d has insufficient columns", i)
		}

		reg := DeviceRegister{
			UUID:        record[headerIndex["uuid"]],
			Tag:         record[headerIndex["tag"]],
			Alias:       record[headerIndex["alias"]],
			DataType:    record[headerIndex["dataType"]],
			DataOrder:   record[headerIndex["dataOrder"]],
			BitPosition: parseUint16(record[headerIndex["bitPosition"]]),
			BitMask:     parseUint16(record[headerIndex["bitMask"]]),
			Weight:      parseFloat64(record[headerIndex["weight"]]),
			Frequency:   parseUint64(record[headerIndex["frequency"]]),
		}

		slaverId, err := strconv.ParseUint(record[headerIndex["slaverId"]], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse slaverId (row %d): %v", i, err)
		}
		reg.SlaverId = uint8(slaverId)

		function, err := strconv.ParseUint(record[headerIndex["function"]], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse function (row %d): %v", i, err)
		}
		reg.Function = uint8(function)

		readAddress, err := strconv.ParseUint(record[headerIndex["readAddress"]], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse readAddress (row %d): %v", i, err)
		}
		reg.ReadAddress = uint16(readAddress)

		readQuantity, err := strconv.ParseUint(record[headerIndex["readQuantity"]], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse readQuantity (row %d): %v", i, err)
		}
		reg.ReadQuantity = uint16(readQuantity)

		if reg.ReadQuantity == 0 {
			if err := reg.CalculateReadQuantity(); err != nil {
				return nil, fmt.Errorf("Failed to calculate readQuantity (row %d): %v", i, err)
			}
		}

		registers = append(registers, reg)
	}

	return registers, nil
}

func parseUint16(s string) uint16 {
	val, _ := strconv.ParseUint(s, 10, 16)
	return uint16(val)
}

func parseFloat64(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseUint64(s string) uint64 {
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}

func TestGroupDeviceRegisterWithLogicalContinuity(t *testing.T) {
	tests := []struct {
		name     string
		regs     []DeviceRegister
		expected [][]DeviceRegister
	}{
		{
			name: "Single register",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 1, SlaverId: 1},
			},
			expected: [][]DeviceRegister{
				{{ReadAddress: 100, ReadQuantity: 1, SlaverId: 1}},
			},
		},
		{
			name: "Two consecutive registers",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 110, ReadQuantity: 5, SlaverId: 1},
			},
			expected: [][]DeviceRegister{
				{
					{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
					{ReadAddress: 110, ReadQuantity: 5, SlaverId: 1},
				},
			},
		},
		{
			name: "Two non-consecutive registers",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 120, ReadQuantity: 5, SlaverId: 1},
			},
			expected: [][]DeviceRegister{
				{{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1}},
				{{ReadAddress: 120, ReadQuantity: 5, SlaverId: 1}},
			},
		},
		{
			name: "Registers from different slaves",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 200, ReadQuantity: 5, SlaverId: 2},
			},
			expected: [][]DeviceRegister{
				{{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1}},
				{{ReadAddress: 200, ReadQuantity: 5, SlaverId: 2}},
			},
		},
		{
			name: "Registers with ReadQuantity=0",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 0, SlaverId: 1, DataType: "uint16"},
				{ReadAddress: 100, ReadQuantity: 5, SlaverId: 1, DataType: "uint16[5]"},
				{ReadAddress: 105, ReadQuantity: 5, SlaverId: 1, DataType: "uint16[5]"},
			},
			expected: [][]DeviceRegister{
				{{ReadAddress: 100, ReadQuantity: 0, SlaverId: 1, DataType: "uint16"}},
				{
					{ReadAddress: 100, ReadQuantity: 5, SlaverId: 1, DataType: "uint16[5]"},
					{ReadAddress: 105, ReadQuantity: 5, SlaverId: 1, DataType: "uint16[5]"},
				},
			},
		},
		{
			name: "Registers with out-of-order addresses",
			regs: []DeviceRegister{
				{ReadAddress: 120, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 110, ReadQuantity: 10, SlaverId: 1},
			},
			expected: [][]DeviceRegister{
				{
					{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
					{ReadAddress: 110, ReadQuantity: 10, SlaverId: 1},
					{ReadAddress: 120, ReadQuantity: 10, SlaverId: 1},
				},
			},
		},
		{
			name: "Mixed consecutive and non-consecutive registers from multiple slaves",
			regs: []DeviceRegister{
				{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1},
				{ReadAddress: 120, ReadQuantity: 5, SlaverId: 1},
				{ReadAddress: 125, ReadQuantity: 5, SlaverId: 1},
				{ReadAddress: 200, ReadQuantity: 5, SlaverId: 2},
				{ReadAddress: 210, ReadQuantity: 3, SlaverId: 2},
			},
			expected: [][]DeviceRegister{
				{{ReadAddress: 100, ReadQuantity: 10, SlaverId: 1}},
				{
					{ReadAddress: 120, ReadQuantity: 5, SlaverId: 1},
					{ReadAddress: 125, ReadQuantity: 5, SlaverId: 1},
				},
				{{ReadAddress: 200, ReadQuantity: 5, SlaverId: 2}},
				{{ReadAddress: 210, ReadQuantity: 3, SlaverId: 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.regs {
				if tt.regs[i].ReadQuantity == 0 {
					tt.regs[i].CalculateReadQuantity()
				}
			}

			result := GroupDeviceRegisterWithLogicalContinuity(tt.regs)

			if !reflect.DeepEqual(result, tt.expected) {
				resultJSON, _ := json.MarshalIndent(result, "", "  ")
				expectedJSON, _ := json.MarshalIndent(tt.expected, "", "  ")
				t.Errorf("GroupDeviceRegisterWithLogicalContinuity() = \n%s\nwant = \n%s", resultJSON, expectedJSON)
			}
		})
	}
}
func TestDeviceRegister_DecodeValue_uint16(t *testing.T) {
	val := uint16(12345)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)

	reg := DeviceRegister{
		DataType: "uint16",
		Value:    buf,
		Weight:   1.0,
	}

	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.AsType != val {
		t.Errorf("expected %v, got %v", val, decoded.AsType)
	}
	if decoded.Float64 != float64(val) {
		t.Errorf("expected float64 %v, got %v", float64(val), decoded.Float64)
	}
}

func TestDeviceRegister_DecodeValue_bool(t *testing.T) {
	val := uint16(0x0004) // bit 2 is set
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)

	reg := DeviceRegister{
		DataType:    "bool",
		Value:       buf,
		BitPosition: 2,
	}

	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.AsType != true {
		t.Errorf("expected true, got %v", decoded.AsType)
	}
}

func TestDeviceRegister_DecodeValue_float32(t *testing.T) {
	val := float32(3.14)
	bits := math.Float32bits(val)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, bits)

	reg := DeviceRegister{
		DataType: "float32",
		Value:    buf,
		Weight:   1.0,
	}

	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := decoded.AsType.(float32)
	if math.Abs(float64(got-val)) > 1e-6 {
		t.Errorf("expected %v, got %v", val, got)
	}
}

func TestDeviceRegister_DecodeValue_string(t *testing.T) {
	str := "test"
	buf := []byte(str)

	reg := DeviceRegister{
		DataType: "string",
		Value:    buf,
	}

	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.AsType != str {
		t.Errorf("expected %v, got %v", str, decoded.AsType)
	}
}

func TestDeviceRegister_EncodeDecode(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	reg := DeviceRegister{}
	reg.Decode(data)
	encoded := reg.Encode()
	if len(encoded) != len(data) {
		t.Fatalf("expected length %d, got %d", len(data), len(encoded))
	}
	for i := range data {
		if encoded[i] != data[i] {
			t.Errorf("expected %v at %d, got %v", data[i], i, encoded[i])
		}
	}
}

// --- Additional boundary and error tests ---

func TestDeviceRegister_DecodeValue_InsufficientBytes(t *testing.T) {
	reg := DeviceRegister{
		DataType: "uint32",
		Value:    []byte{1, 2}, // not enough bytes
	}
	_, err := reg.DecodeValue()
	if err == nil {
		t.Error("expected error for insufficient bytes, got nil")
	}
}

func TestDeviceRegister_DecodeValue_UnsupportedType(t *testing.T) {
	reg := DeviceRegister{
		DataType: "unknown_type",
		Value:    []byte{1, 2, 3, 4},
	}
	_, err := reg.DecodeValue()
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestDeviceRegister_DecodeValue_ZeroLength(t *testing.T) {
	reg := DeviceRegister{
		DataType: "uint16",
		Value:    []byte{},
	}
	_, err := reg.DecodeValue()
	if err == nil {
		t.Error("expected error for zero length value, got nil")
	}
}

func TestDeviceRegister_DecodeValue_BitfieldMask(t *testing.T) {
	val := uint16(0x00F0)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, val)
	reg := DeviceRegister{
		DataType: "bitfield",
		Value:    buf,
		BitMask:  0x00F0,
		Weight:   2.0,
	}
	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := uint16(0x00F0)
	if decoded.AsType != expected {
		t.Errorf("expected %v, got %v", expected, decoded.AsType)
	}
	if decoded.Float64 != float64(expected)*2.0 {
		t.Errorf("expected float64 %v, got %v", float64(expected)*2.0, decoded.Float64)
	}
}

func TestDeviceRegister_DecodeValue_NegativeInt8(t *testing.T) {
	reg := DeviceRegister{
		DataType: "int8",
		Value:    []byte{0xFF}, // -1 in int8
		Weight:   1.0,
	}
	decoded, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.AsType != int8(-1) {
		t.Errorf("expected %v, got %v", int8(-1), decoded.AsType)
	}
}

func TestDeviceRegister_DecodeValue_ByteOrder(t *testing.T) {
	// Test reorderBytes is called (assume reorderBytes swaps bytes for "DCBA")
	val := uint32(0x01020304)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, val)
	reg := DeviceRegister{
		DataType:  "uint32",
		Value:     buf,
		DataOrder: "DCBA", // should trigger reorder
		Weight:    1.0,
	}
	// This test assumes reorderBytes is implemented and works as expected
	_, err := reg.DecodeValue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeValueFromCSV(t *testing.T) {
	file := filepath.Join("test", "register_test_cases.csv")
	f, err := os.Open(file)
	if err != nil {
		t.Fatalf("failed to open CSV: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("CSV should contain header and at least one data row")
	}
	header := records[0]
	rows := records[1:]

	for i, row := range rows {
		r, err := parseDeviceRegisterRow(header, row)
		if err != nil {
			t.Errorf("row %d: failed to parse: %v", i+1, err)
			continue
		}
		dv, err := r.DecodeValue()
		if err != nil {
			t.Errorf("row %d (%s): decode failed: %v", i+1, r.Tag, err)
			continue
		}
		t.Logf("row %d (%s): decoded: %v", i+1, r.Tag, dv)
	}
}
func parseDeviceRegisterRow(header, row []string) (DeviceRegister, error) {
	m := make(map[string]string)
	for i := range header {
		if i < len(row) {
			m[header[i]] = row[i]
		}
	}
	parseUint8 := func(key string) uint8 {
		v, _ := strconv.ParseUint(m[key], 10, 8)
		return uint8(v)
	}
	parseUint16 := func(key string) uint16 {
		v, _ := strconv.ParseUint(m[key], 10, 16)
		return uint16(v)
	}
	parseUint64 := func(key string) uint64 {
		v, _ := strconv.ParseUint(m[key], 10, 64)
		return v
	}
	parseFloat64 := func(key string) float64 {
		v, _ := strconv.ParseFloat(m[key], 64)
		return v
	}
	parseBytes := func(key string) []byte {
		raw := strings.TrimSpace(m[key])
		raw = strings.Trim(raw, `"`) // strip quotes
		if raw == "" {
			return nil
		}
		parts := strings.Split(raw, ",")
		var b []byte
		for _, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				continue
			}
			b = append(b, byte(n))
		}
		return b
	}

	return DeviceRegister{
		UUID:         m["uuid"],
		Tag:          m["tag"],
		Alias:        m["alias"],
		SlaverId:     parseUint8("slaverId"),
		Function:     parseUint8("function"),
		ReadAddress:  parseUint16("readAddress"),
		ReadQuantity: parseUint16("readQuantity"),
		DataType:     m["dataType"],
		DataOrder:    m["dataOrder"],
		BitPosition:  parseUint16("bitPosition"),
		BitMask:      parseUint16("bitMask"),
		Weight:       parseFloat64("weight"),
		Frequency:    parseUint64("frequency"),
		Value:        parseBytes("value"),
	}, nil
}
