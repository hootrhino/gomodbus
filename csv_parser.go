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
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// isValidByteOrder checks whether the given byte order is valid.
func isValidByteOrder(order string) bool {
	validOrders := map[string]struct{}{
		"A": {}, "AB": {}, "BA": {}, "ABCD": {}, "DCBA": {},
		"BADC": {}, "CDAB": {}, "ABCDEFGH": {}, "HGFEDCBA": {},
		"BADCFEHG": {}, "GHEFCDAB": {},
	}
	_, ok := validOrders[order]
	return ok
}

// CSVRegisterParser handles conversion between CSV and DeviceRegister
type CSVRegisterParser struct {
	// CSV headers mapping
	headers []string
}

// NewCSVRegisterParser creates a new CSV register parser
func NewCSVRegisterParser() *CSVRegisterParser {
	return &CSVRegisterParser{
		headers: []string{
			"uuid",
			"tag",
			"alias",
			"slaverId",
			"function",
			"readAddress",
			"readQuantity",
			"dataType",
			"dataOrder",
			"bitPosition",
			"bitMask",
			"weight",
			"frequency",
		},
	}
}

// ParseCSV parses CSV data and returns a slice of DeviceRegister
func (p *CSVRegisterParser) ParseCSV(reader io.Reader) ([]DeviceRegister, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// Parse header row
	header := records[0]
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	// Validate required fields in header
	requiredFields := []string{"uuid", "tag", "slaverId", "function", "readAddress", "dataType"}
	for _, field := range requiredFields {
		if _, exists := headerMap[field]; !exists {
			return nil, fmt.Errorf("missing required field in CSV header: %s", field)
		}
	}

	// Parse data rows
	var registers []DeviceRegister
	for i, record := range records[1:] {
		register, err := p.parseRegisterFromRecord(record, headerMap, i+2)
		if err != nil {
			return nil, fmt.Errorf("error parsing row %d: %w", i+2, err)
		}

		// Validate the parsed register for consistency and Modbus rules
		if err := p.ValidateRegister(register); err != nil {
			return nil, fmt.Errorf("validation error for row %d (UUID: %s, Tag: %s): %w", i+2, register.UUID, register.Tag, err)
		}

		registers = append(registers, register)
	}

	return registers, nil
}

// parseRegisterFromRecord parses a single CSV record into a DeviceRegister
func (p *CSVRegisterParser) parseRegisterFromRecord(record []string, headerMap map[string]int, rowNum int) (DeviceRegister, error) {
	var register DeviceRegister

	// Helper function to get field value
	getField := func(fieldName string) string {
		if idx, exists := headerMap[fieldName]; exists && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	// Helper function for parsing unsigned integers
	parseUintField := func(fieldName string, bitSize int) (uint64, error) {
		strVal := getField(fieldName)
		if strVal == "" {
			return 0, fmt.Errorf("'%s' is required", fieldName)
		}
		val, err := strconv.ParseUint(strVal, 10, bitSize)
		if err != nil {
			return 0, fmt.Errorf("invalid '%s': %w", fieldName, err)
		}
		return val, nil
	}

	// Helper function for parsing float
	parseFloatField := func(fieldName string, bitSize int) (float64, error) {
		strVal := getField(fieldName)
		if strVal == "" {
			return 0.0, fmt.Errorf("'%s' is required", fieldName)
		}
		val, err := strconv.ParseFloat(strVal, bitSize)
		if err != nil {
			return 0.0, fmt.Errorf("invalid '%s': %w", fieldName, err)
		}
		return val, nil
	}

	// Parse UUID (required)
	register.UUID = getField("uuid")
	if register.UUID == "" {
		return register, fmt.Errorf("'UUID' is required at row %d", rowNum)
	}

	// Parse Tag (required)
	register.Tag = getField("tag")
	if register.Tag == "" {
		return register, fmt.Errorf("'Tag' is required at row %d", rowNum)
	}

	// Parse Alias (optional)
	register.Alias = getField("alias")

	// Parse SlaveId (required)
	slaveIdVal, err := parseUintField("slaverId", 8)
	if err != nil {
		return register, fmt.Errorf("at row %d: %w", rowNum, err)
	}
	register.SlaverId = uint8(slaveIdVal)

	// Parse Function (required)
	functionVal, err := parseUintField("function", 8)
	if err != nil {
		return register, fmt.Errorf("at row %d: %w", rowNum, err)
	}
	register.Function = uint8(functionVal)

	// Parse ReadAddress (required)
	readAddressVal, err := parseUintField("readAddress", 16)
	if err != nil {
		return register, fmt.Errorf("at row %d: %w", rowNum, err)
	}
	register.ReadAddress = uint16(readAddressVal)

	// Parse ReadQuantity (optional, will be calculated if not provided)
	readQuantityStr := getField("readQuantity")
	if readQuantityStr != "" {
		readQuantity, err := strconv.ParseUint(readQuantityStr, 10, 16)
		if err != nil {
			return register, fmt.Errorf("invalid ReadQuantity at row %d: %w", rowNum, err)
		}
		register.ReadQuantity = uint16(readQuantity)
	}

	// Parse DataType (required)
	register.DataType = getField("dataType")
	if register.DataType == "" {
		return register, fmt.Errorf("'DataType' is required at row %d", rowNum)
	}

	// Auto-calculate ReadQuantity if not provided in CSV
	if register.ReadQuantity == 0 {
		quantity, err := register.CalculateReadQuantity()
		if err != nil {
			return register, fmt.Errorf("failed to calculate ReadQuantity for DataType '%s' at row %d: %w", register.DataType, rowNum, err)
		}
		register.ReadQuantity = quantity
	}

	// Parse DataOrder (optional, default to "ABCD")
	register.DataOrder = getField("dataOrder")
	if register.DataOrder == "" {
		register.DataOrder = "ABCD" // Default value
	}
	if !isValidByteOrder(register.DataOrder) {
		return register, fmt.Errorf("invalid 'DataOrder' '%s' at row %d", register.DataOrder, rowNum)
	}

	// Parse BitPosition (0-15, default to 0)
	bitPositionStr := getField("bitPosition")
	if bitPositionStr != "" {
		bitPosition, err := strconv.ParseUint(bitPositionStr, 10, 16)
		if err != nil {
			return register, fmt.Errorf("invalid 'BitPosition' at row %d: %w", rowNum, err)
		}
		if bitPosition > 15 { // Stricter check for 0-15 range
			return register, fmt.Errorf("'BitPosition' at row %d must be 0-15", rowNum)
		}
		register.BitPosition = uint16(bitPosition)
	} else {
		register.BitPosition = 0 // Explicitly set default
	}

	// Parse BitMask (optional, default to 0x01)
	bitMaskStr := getField("bitMask")
	if bitMaskStr != "" {
		var bitMask uint64
		// Support both hex (0x01) and decimal (1) formats
		if strings.HasPrefix(bitMaskStr, "0x") || strings.HasPrefix(bitMaskStr, "0X") {
			bitMask, err = strconv.ParseUint(bitMaskStr, 0, 16)
		} else {
			bitMask, err = strconv.ParseUint(bitMaskStr, 10, 16)
		}
		if err != nil {
			return register, fmt.Errorf("invalid BitMask at row %d: %w", rowNum, err)
		}
		register.BitMask = uint16(bitMask)
	} else {
		register.BitMask = 0x01 // Default value
	}

	// Parse Weight (optional, default to 1.0)
	weightStr := getField("weight")
	if weightStr != "" {
		weight, err := parseFloatField("weight", 64)
		if err != nil {
			return register, fmt.Errorf("at row %d: %w", rowNum, err)
		}
		register.Weight = weight
	} else {
		register.Weight = 1.0 // Default value
	}

	// Parse Frequency (optional, default to 1000)
	frequencyStr := getField("frequency")
	if frequencyStr != "" {
		frequency, err := parseUintField("frequency", 64)
		if err != nil {
			return register, fmt.Errorf("at row %d: %w", rowNum, err)
		}
		register.Frequency = frequency
	} else {
		register.Frequency = 1000 // Default value, consistent with test
	}

	return register, nil
}

// ToCSV converts a slice of DeviceRegister to CSV format
func (p *CSVRegisterParser) ToCSV(registers []DeviceRegister, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(p.headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, register := range registers {
		record, err := p.registerToRecord(register)
		if err != nil {
			return fmt.Errorf("failed to convert register %s to CSV record: %w", register.Tag, err)
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record for register %s: %w", register.Tag, err)
		}
	}

	return nil
}

// registerToRecord converts a DeviceRegister to CSV record
func (p *CSVRegisterParser) registerToRecord(register DeviceRegister) ([]string, error) {
	// Convert BitMask to hex string (e.g., 0x01, 0xFF)
	bitMaskStr := fmt.Sprintf("0x%04X", register.BitMask) // Use %04X for 16-bit mask, like 0x0001, 0xFFFF

	return []string{
		register.UUID,
		register.Tag,
		register.Alias,
		strconv.FormatUint(uint64(register.SlaverId), 10),
		strconv.FormatUint(uint64(register.Function), 10),
		strconv.FormatUint(uint64(register.ReadAddress), 10),
		strconv.FormatUint(uint64(register.ReadQuantity), 10),
		register.DataType,
		register.DataOrder,
		strconv.FormatUint(uint64(register.BitPosition), 10),
		bitMaskStr,
		strconv.FormatFloat(register.Weight, 'f', -1, 64), // -1 uses smallest number of digits necessary
		strconv.FormatUint(register.Frequency, 10),
	}, nil
}

// ValidateRegister validates a DeviceRegister for common issues and data consistency.
func (p *CSVRegisterParser) ValidateRegister(register DeviceRegister) error {
	// Check required fields (basic check, more robust parsing ensures this)
	if register.UUID == "" {
		return fmt.Errorf("'UUID' is required")
	}
	if register.Tag == "" {
		return fmt.Errorf("'Tag' is required")
	}
	if register.DataType == "" {
		return fmt.Errorf("'DataType' is required")
	}

	// Validate Modbus function codes
	switch register.Function {
	case 1, 2, 3, 4, 5, 6, 15, 16:
		// Valid Modbus function codes
	default:
		return fmt.Errorf("invalid Modbus function code: %d", register.Function)
	}

	// Validate data type and get expected register quantity
	_, expectedRegisters, err := parseArrayType(register.DataType)
	if err != nil {
		return fmt.Errorf("invalid DataType '%s': %w", register.DataType, err)
	}

	// Stricter ReadQuantity validation: Ensure it matches the expected quantity for the DataType
	if register.ReadQuantity != uint16(expectedRegisters) {
		return fmt.Errorf("ReadQuantity %d does not match expected quantity %d for DataType '%s'",
			register.ReadQuantity, expectedRegisters, register.DataType)
	}

	// Validate data order
	if register.DataOrder == "" {
		return fmt.Errorf("'DataOrder' is required (defaults to ABCD if not provided, but was empty)")
	}
	if !isValidByteOrder(register.DataOrder) {
		return fmt.Errorf("invalid DataOrder: '%s'", register.DataOrder)
	}

	// Validate bit position and bit mask for boolean and bitfield types
	if register.DataType == "bool" || register.DataType == "bitfield" {
		if register.BitPosition > 15 {
			return fmt.Errorf("BitPosition must be 0-15 for %s type", register.DataType)
		}
	} else {
		// For non-boolean/bitfield types, BitPosition should typically be 0
		if register.BitPosition != 0 {
			return fmt.Errorf("BitPosition must be 0 for DataType '%s'", register.DataType)
		}
	}

	return nil
}

// ParseCSVFromString parses CSV data from a string
func (p *CSVRegisterParser) ParseCSVFromString(csvData string) ([]DeviceRegister, error) {
	reader := strings.NewReader(csvData)
	return p.ParseCSV(reader)
}

// ToCSVString converts registers to CSV string
func (p *CSVRegisterParser) ToCSVString(registers []DeviceRegister) (string, error) {
	var builder strings.Builder
	err := p.ToCSV(registers, &builder)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}
