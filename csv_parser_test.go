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
	"strings"
	"testing"
)

// TestCSVRegisterParser_ParseCSV tests basic CSV parsing functionality
func TestCSVRegisterParser_ParseCSV(t *testing.T) {
	parser := NewCSVRegisterParser()

	csvData := `uuid,tag,alias,slaverId,function,readAddress,readQuantity,dataType,dataOrder,bitPosition,bitMask,weight,frequency
test-uuid-1,TAG001,Temperature Sensor,1,3,1000,1,uint16,ABCD,0,0x01,0.1,1000
test-uuid-2,TAG002,Pressure Sensor,1,3,1002,2,float32,DCBA,0,0x00,1.0,2000
test-uuid-3,TAG003,Boolean Status,1,3,1006,1,bool,ABCD,5,0x20,1.0,500
test-uuid-4,TAG004,Default Check,1,4,2000,,uint16,,0,,1.0,` // Missing readQuantity, dataOrder, bitPosition, bitMask, frequency (should use defaults/calculate)

	registers, err := parser.ParseCSVFromString(csvData)
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(registers) != 4 {
		t.Fatalf("Expected 4 registers, got %d", len(registers))
	}
	t.Logf("Parsed Registers: %+v", registers)

	// Verify defaults and calculated values for test-uuid-4
	reg4 := registers[3]
	if reg4.UUID != "test-uuid-4" {
		t.Errorf("Expected UUID test-uuid-4, got %s", reg4.UUID)
	}
	if reg4.ReadQuantity != 1 { // uint16 expects 1 register
		t.Errorf("Expected ReadQuantity 1 for TAG004, got %d", reg4.ReadQuantity)
	}
	if reg4.DataOrder != "ABCD" {
		t.Errorf("Expected DataOrder ABCD for TAG004, got %s", reg4.DataOrder)
	}
	if reg4.BitPosition != 0 {
		t.Errorf("Expected BitPosition 0 for TAG004, got %d", reg4.BitPosition)
	}
	if reg4.BitMask != 0x01 {
		t.Errorf("Expected BitMask 0x01 for TAG004, got 0x%X", reg4.BitMask)
	}
	if reg4.Frequency != 1000 {
		t.Errorf("Expected Frequency 1000 for TAG004, got %d", reg4.Frequency)
	}
}

// TestCSVRegisterParser_ValidationErrors tests various validation scenarios
func TestCSVRegisterParser_ValidationErrors(t *testing.T) {
	parser := NewCSVRegisterParser()

	tests := []struct {
		name    string
		csv     string
		wantErr string
	}{
		{
			name: "Missing UUID in row",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
,TAG001,1,3,1000,uint16`,
			wantErr: "'UUID' is required at row 2",
		},
		{
			name: "Missing Tag in row",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
uuid-1,,1,3,1000,uint16`,
			wantErr: "'Tag' is required at row 2",
		},
		{
			name: "Missing SlaveId in row",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
uuid-1,TAG001,,3,1000,uint16`,
			wantErr: "error parsing row 2: at row 2: 'slaverId' is required", // Adjusted to match wrapped error
		},
		{
			name: "Invalid SlaveId format",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
uuid-1,TAG001,abc,3,1000,uint16`,
			wantErr: "error parsing row 2: at row 2: invalid 'slaverId': strconv.ParseUint: parsing \"abc\": invalid syntax", // Adjusted to match wrapped error
		},
		{
			name: "Invalid BitMask hex format",
			csv: `uuid,tag,slaverId,function,readAddress,dataType,bitMask
uuid-1,TAG001,1,3,1000,uint16,0xZZ`,
			wantErr: "invalid BitMask at row 2",
		},
		{
			name: "Invalid DataOrder format",
			csv: `uuid,tag,slaverId,function,readAddress,dataType,dataOrder
uuid-1,TAG001,1,3,1000,uint16,ZZZZ`,
			wantErr: "invalid 'DataOrder' 'ZZZZ' at row 2",
		},
		{
			name: "BitPosition out of range (0-15)", // Stricter check
			csv: `uuid,tag,slaverId,function,readAddress,dataType,bitPosition
uuid-1,TAG001,1,3,1000,bool,16`,
			wantErr: "'BitPosition' at row 2 must be 0-15",
		},
		{
			name: "Invalid Modbus Function code",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
uuid-1,TAG001,1,99,1000,uint16`, // Function 99 is invalid
			wantErr: "invalid Modbus function code: 99",
		},
		{
			name: "Unsupported DataType",
			csv: `uuid,tag,slaverId,function,readAddress,dataType
uuid-1,TAG001,1,3,1000,unsupportedType`,
			wantErr: "error parsing row 2: failed to calculate ReadQuantity for DataType 'unsupportedType' at row 2: unsupported data type: unsupportedType", // Adjusted to match wrapped error
		},
		{
			name: "ReadQuantity mismatch with DataType",
			csv: `uuid,tag,slaverId,function,readAddress,readQuantity,dataType
uuid-1,TAG001,1,3,1000,1,float32`, // float32 expects 2 registers, but 1 is given
			wantErr: "ReadQuantity 1 does not match expected quantity 2 for DataType 'float32'",
		},
		{
			name: "BitPosition not zero for non-boolean type",
			csv: `uuid,tag,slaverId,function,readAddress,dataType,bitPosition
uuid-1,TAG001,1,3,1000,uint16,1`, // uint16 with BitPosition 1, should fail
			wantErr: "BitPosition must be 0 for DataType 'uint16'",
		},
		{
			name: "Missing required header field",
			csv: `uuid,tag,function,readAddress,dataType
uuid-1,TAG001,3,1000,uint16`, // Missing slaverId in header
			wantErr: "missing required field in CSV header: slaverId",
		},
		{
			name:    "Empty CSV file content",
			csv:     ``,
			wantErr: "empty CSV file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseCSVFromString(tt.csv)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestCSVRegisterParser_EmptyDataRows tests scenario with header but no data rows
func TestCSVRegisterParser_EmptyDataRows(t *testing.T) {
	parser := NewCSVRegisterParser()
	csvData := `uuid,tag,slaverId,function,readAddress,dataType`

	registers, err := parser.ParseCSVFromString(csvData)
	if err != nil {
		t.Fatalf("Expected no error for CSV with only header, got %v", err)
	}
	if len(registers) != 0 {
		t.Errorf("Expected 0 registers for CSV with only header, got %d", len(registers))
	}
}

func TestCSVRegisterParser_ToCSV(t *testing.T) {
	parser := NewCSVRegisterParser()

	registers := []DeviceRegister{
		{
			UUID:         "uuid-123",
			Tag:          "T1",
			Alias:        "Sensor",
			SlaverId:     1,
			Function:     3,
			ReadAddress:  100,
			ReadQuantity: 2,
			DataType:     "float32",
			DataOrder:    "ABCD",
			BitPosition:  0,
			BitMask:      0xFF,
			Weight:       1.0,
			Frequency:    500,
		},
		{
			UUID:         "uuid-456",
			Tag:          "T2",
			Alias:        "", // Empty alias
			SlaverId:     2,
			Function:     4,
			ReadAddress:  200,
			ReadQuantity: 1,
			DataType:     "bool",
			DataOrder:    "A",
			BitPosition:  7,
			BitMask:      0x80, // 1 << 7
			Weight:       0.5,
			Frequency:    100,
		},
	}

	csvStr, err := parser.ToCSVString(registers)
	if err != nil {
		t.Fatalf("ToCSVString failed: %v", err)
	}

	t.Logf("Generated CSV:\n%s", csvStr)

	// Basic checks for content
	if !strings.Contains(csvStr, "uuid-123") {
		t.Errorf("Expected CSV to contain register UUID uuid-123")
	}
	if !strings.Contains(csvStr, "T1") {
		t.Errorf("Expected CSV to contain register Tag T1")
	}
	if !strings.Contains(csvStr, "float32") {
		t.Errorf("Expected CSV to contain DataType float32")
	}
	if !strings.Contains(csvStr, "0x00FF") { // Check BitMask format
		t.Errorf("Expected CSV to contain BitMask 0x00FF, got %s", csvStr)
	}
	if !strings.Contains(csvStr, "0.5") {
		t.Errorf("Expected CSV to contain Weight 0.5")
	}
	if !strings.Contains(csvStr, "100") { // Check Frequency for T2
		t.Errorf("Expected CSV to contain Frequency 100")
	}

	// Try parsing back the generated CSV to ensure consistency
	parsedRegisters, err := parser.ParseCSVFromString(csvStr)
	if err != nil {
		t.Fatalf("Failed to parse generated CSV: %v", err)
	}

	if len(parsedRegisters) != len(registers) {
		t.Fatalf("Expected %d registers after parsing generated CSV, got %d", len(registers), len(parsedRegisters))
	}

	// Deep comparison (optional but good practice)
	// For simplicity, let's just check a few key fields for the first register
	if parsedRegisters[0].UUID != registers[0].UUID ||
		parsedRegisters[0].Tag != registers[0].Tag ||
		parsedRegisters[0].ReadQuantity != registers[0].ReadQuantity ||
		parsedRegisters[0].DataType != registers[0].DataType ||
		parsedRegisters[0].BitMask != registers[0].BitMask {
		t.Errorf("Mismatch after round-trip CSV conversion for first register.\nExpected: %+v\nGot: %+v", registers[0], parsedRegisters[0])
	}
}

// TestParseArrayType tests the helper function parseArrayType
func TestParseArrayType(t *testing.T) {
	tests := []struct {
		dataType      string
		expectedBytes int
		expectedRegs  int
		wantErr       bool
	}{
		{"bool", 1, 1, false},
		{"uint16", 2, 1, false},
		{"int16", 2, 1, false},
		{"float32", 4, 2, false},
		{"uint32", 4, 2, false},
		{"float64", 8, 4, false},
		{"unsupported", 0, 0, true},
		{"", 0, 0, true}, // Empty data type should also be an error
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			bytes, regs, err := parseArrayType(tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArrayType(%q) error status mismatch: got err %v, wantErr %t", tt.dataType, err, tt.wantErr)
			}
			if regs != tt.expectedBytes {
				t.Errorf("parseArrayType(%q) expected bytes %d, got %s", tt.dataType, tt.expectedBytes, bytes)
			}
			if regs != tt.expectedRegs {
				t.Errorf("parseArrayType(%q) expected registers %d, got %d", tt.dataType, tt.expectedRegs, regs)
			}
		})
	}
}

// TestCalculateReadQuantity tests the CalculateReadQuantity method on DeviceRegister
func TestDeviceRegister_CalculateReadQuantity(t *testing.T) {
	tests := []struct {
		name          string
		dataType      string
		expectedReadQ uint16
		wantErr       bool
	}{
		{"bool type", "bool", 1, false},
		{"uint16 type", "uint16", 1, false},
		{"float32 type", "float32", 2, false},
		{"unsupported type", "badType", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := DeviceRegister{DataType: tt.dataType}
			quantity, err := reg.CalculateReadQuantity()
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateReadQuantity for %s error status mismatch: got err %v, wantErr %t", tt.dataType, err, tt.wantErr)
			}
			if quantity != tt.expectedReadQ {
				t.Errorf("CalculateReadQuantity for %s expected %d, got %d", tt.dataType, tt.expectedReadQ, quantity)
			}
		})
	}
}
