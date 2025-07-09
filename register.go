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
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unsafe"
)

// DeviceRegister represents a Modbus register with metadata
type DeviceRegister struct {
	UUID         string  `json:"uuid"`         // Unique identifier for the register
	Tag          string  `json:"tag"`          // A unique identifier or label for the register
	Alias        string  `json:"alias"`        // A human-readable name or alias for the register
	SlaverId     uint8   `json:"slaverId"`     // ID of the Modbus slave device
	Function     uint8   `json:"function"`     // Modbus function code (e.g., 3 for Read Holding Registers)
	ReadAddress  uint16  `json:"readAddress"`  // Address of the register in the Modbus device
	ReadQuantity uint16  `json:"readQuantity"` // Number of registers to read/write
	DataType     string  `json:"dataType"`     // Data type of the register value (e.g., uint16, int32, float32,uint16[1], int32[2], float32[6] ...)
	DataOrder    string  `json:"dataOrder"`    // Byte order for multi-byte values (e.g., ABCD, DCBA)
	BitPosition  uint16  `json:"bitPosition"`  // Bit position for bit-level operations (e.g., 0, 1, 2)
	BitMask      uint16  `json:"bitMask"`      // Bitmask for bit-level operations (e.g., 0x01, 0x02)
	Weight       float64 `json:"weight"`       // Scaling factor for the register value
	Frequency    uint64  `json:"frequency"`    // Polling frequency in milliseconds
	Value        []byte  `json:"value"`        // Raw value of the register as a byte array (variable length)
	Status       string  `json:"status"`       // Status of the register (e.g., "OK", "ERROR:Reason Msg")
}

// CalculateReadQuantity calculates the ReadQuantity based on the DataType
func (r *DeviceRegister) CalculateReadQuantity() (uint16, error) {
	baseType, count, err := parseArrayType(r.DataType)
	if err != nil {
		return 0, err
	}

	requiredBytesPerElement, err := getRequiredBytes(baseType)
	if err != nil {
		return 0, err
	}

	// Calculate required number of registers (each register is 2 bytes)
	r.ReadQuantity = uint16(count * (requiredBytesPerElement / 2))
	return r.ReadQuantity, nil
}

// DecodeValue decodes the raw register value based on its data type
// Supports both single values and arrays (e.g., uint16[10], float32[5])
func (r DeviceRegister) DecodeValue() (DecodedValue, error) {
	// Initialize result with raw value
	result := DecodedValue{
		Raw:  r.Value,
		Type: r.DataType,
	}

	// Handle empty value case
	if len(r.Value) == 0 {
		return result, fmt.Errorf("empty value for register %s", r.Tag)
	}

	// Parse data type to get base type and count
	baseType, count, err := parseArrayType(r.DataType)
	if err != nil {
		return result, fmt.Errorf("invalid data type %s for register %s: %w", r.DataType, r.Tag, err)
	}

	// Get required bytes per element
	requiredBytesPerElement, err := getRequiredBytes(baseType)
	if err != nil {
		return result, fmt.Errorf("unsupported base type %s for register %s: %w", baseType, r.Tag, err)
	}

	// Auto-calculate array length if needed
	if count == 0 {
		if requiredBytesPerElement == 0 {
			return result, fmt.Errorf("cannot auto-calculate array length for variable-length type %s", baseType)
		}
		totalBytes := int(r.ReadQuantity) * 2 // Each register is 2 bytes
		count = totalBytes / requiredBytesPerElement
		if count <= 0 {
			return result, fmt.Errorf("ReadQuantity %d too small for %s[] (need at least %d bytes)",
				r.ReadQuantity, baseType, requiredBytesPerElement)
		}
	}

	// Validate data length for non-string types
	if baseType != "string" {
		totalRequired := requiredBytesPerElement * count
		if len(r.Value) < totalRequired {
			return result, fmt.Errorf("insufficient data for %s[%d]: have %d bytes, need %d",
				baseType, count, len(r.Value), totalRequired)
		}
	}

	// Handle array types
	if count > 1 {
		return r.decodeArrayValue(result, baseType, count, requiredBytesPerElement)
	}

	// Handle single value types
	return r.decodeSingleValue(result, baseType)
}

// decodeArrayValue handles decoding of array types
func (r DeviceRegister) decodeArrayValue(result DecodedValue, baseType string, count, bytesPerElement int) (DecodedValue, error) {
	typeMap := map[string]reflect.Type{
		"byte":    reflect.TypeOf(uint8(0)),
		"uint8":   reflect.TypeOf(uint8(0)),
		"int8":    reflect.TypeOf(int8(0)),
		"uint16":  reflect.TypeOf(uint16(0)),
		"int16":   reflect.TypeOf(int16(0)),
		"uint32":  reflect.TypeOf(uint32(0)),
		"int32":   reflect.TypeOf(int32(0)),
		"uint64":  reflect.TypeOf(uint64(0)),
		"int64":   reflect.TypeOf(int64(0)),
		"float32": reflect.TypeOf(float32(0)),
		"float64": reflect.TypeOf(float64(0)),
	}

	elemType, ok := typeMap[baseType]
	if !ok {
		return result, fmt.Errorf("unsupported base type: %s", baseType)
	}

	values := reflect.MakeSlice(reflect.SliceOf(elemType), 0, count)
	var sum float64

	for i := 0; i < count; i++ {
		offset := i * bytesPerElement

		// Bounds check
		if offset+bytesPerElement > len(r.Value) {
			return result, fmt.Errorf("array element %d out of bounds for %s[%d]", i, baseType, count)
		}

		// Get element bytes and reorder if necessary
		elementBytes := r.Value[offset : offset+bytesPerElement]
		if len(elementBytes) > 1 {
			elementBytes = reorderBytes(elementBytes, r.DataOrder)
		}

		// Decode element based on base type
		val, err := r.decodeElementValue(elementBytes, baseType)
		if err != nil {
			return result, fmt.Errorf("failed to decode array element %d: %w", i, err)
		}

		values = reflect.Append(values, reflect.ValueOf(val))
		sum += convertToFloat64(val)
	}

	result.AsType = values.Interface()
	result.Float64 = sum * r.Weight
	return result, nil
}

// decodeSingleValue handles decoding of single value types
func (r DeviceRegister) decodeSingleValue(result DecodedValue, baseType string) (DecodedValue, error) {
	// Reorder bytes if necessary
	bytes := r.Value
	if len(bytes) > 1 {
		bytes = reorderBytes(bytes, r.DataOrder)
	}

	// Handle special cases first
	switch baseType {
	case "bitfield":
		return r.decodeBitfield(result, bytes)
	case "bool":
		return r.decodeBool(result, bytes)
	case "string":
		return r.decodeString(result, bytes)
	}

	// Handle numeric types
	val, err := r.decodeElementValue(bytes, baseType)
	if err != nil {
		return result, err
	}

	result.AsType = val
	result.Float64 = convertToFloat64(val) * r.Weight
	return result, nil
}

// decodeElementValue decodes a single element of the given base type
func (r DeviceRegister) decodeElementValue(bytes []byte, baseType string) (any, error) {
	switch baseType {
	case "byte", "uint8":
		if len(bytes) < 1 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 1, have %d", baseType, len(bytes))
		}
		return bytes[0], nil

	case "int8":
		if len(bytes) < 1 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 1, have %d", baseType, len(bytes))
		}
		return int8(bytes[0]), nil

	case "uint16":
		if len(bytes) < 2 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 2, have %d", baseType, len(bytes))
		}
		return binary.BigEndian.Uint16(bytes), nil

	case "int16":
		if len(bytes) < 2 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 2, have %d", baseType, len(bytes))
		}
		return int16(binary.BigEndian.Uint16(bytes)), nil

	case "uint32":
		if len(bytes) < 4 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 4, have %d", baseType, len(bytes))
		}
		return binary.BigEndian.Uint32(bytes), nil

	case "int32":
		if len(bytes) < 4 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 4, have %d", baseType, len(bytes))
		}
		return int32(binary.BigEndian.Uint32(bytes)), nil

	case "uint64":
		if len(bytes) < 8 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 8, have %d", baseType, len(bytes))
		}
		return binary.BigEndian.Uint64(bytes), nil

	case "int64":
		if len(bytes) < 8 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 8, have %d", baseType, len(bytes))
		}
		return int64(binary.BigEndian.Uint64(bytes)), nil

	case "float32":
		if len(bytes) < 4 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 4, have %d", baseType, len(bytes))
		}
		return float32FromBits(binary.BigEndian.Uint32(bytes)), nil

	case "float64":
		if len(bytes) < 8 {
			return nil, fmt.Errorf("insufficient bytes for %s: need 8, have %d", baseType, len(bytes))
		}
		return float64FromBits(binary.BigEndian.Uint64(bytes)), nil

	default:
		return nil, fmt.Errorf("unsupported element type: %s", baseType)
	}
}

// decodeBitfield handles bitfield decoding
func (r DeviceRegister) decodeBitfield(result DecodedValue, bytes []byte) (DecodedValue, error) {
	if len(bytes) < 2 {
		return result, fmt.Errorf("insufficient bytes for bitfield: need 2, have %d", len(bytes))
	}

	val := binary.BigEndian.Uint16(bytes[:2]) & r.BitMask
	result.AsType = val
	result.Float64 = float64(val) * r.Weight
	return result, nil
}

// decodeBool handles boolean decoding
func (r DeviceRegister) decodeBool(result DecodedValue, bytes []byte) (DecodedValue, error) {
	if len(bytes) < 2 {
		return result, fmt.Errorf("insufficient bytes for bool: need 2, have %d", len(bytes))
	}

	val := binary.BigEndian.Uint16(bytes[:2])
	b := CheckBit(val, r.BitPosition)
	result.AsType = b
	result.Float64 = 0.0
	if b {
		result.Float64 = 1.0
	}
	return result, nil
}

// decodeString handles string decoding
func (r DeviceRegister) decodeString(result DecodedValue, bytes []byte) (DecodedValue, error) {
	// Remove null terminators and trim whitespace
	str := string(bytes)
	if nullIndex := strings.IndexByte(str, 0); nullIndex != -1 {
		str = str[:nullIndex]
	}
	str = strings.TrimSpace(str)

	result.AsType = str
	result.Float64 = 0.0
	return result, nil
}

// convertToFloat64 converts various numeric types to float64 for summation
func convertToFloat64(val any) float64 {
	switch v := val.(type) {
	case uint8:
		return float64(v)
	case int8:
		return float64(v)
	case uint16:
		return float64(v)
	case int16:
		return float64(v)
	case uint32:
		return float64(v)
	case int32:
		return float64(v)
	case uint64:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0.0
	}
}

// Enhanced getRequiredBytes with support for more types
func getRequiredBytes(dataType string) (int, error) {
	switch dataType {
	case "byte", "uint8", "int8":
		return 1, nil
	case "bool", "bitfield", "uint16", "int16":
		return 2, nil
	case "uint32", "int32", "float32":
		return 4, nil
	case "uint64", "int64", "float64":
		return 8, nil
	case "string":
		// String type has variable length
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown data type: %s", dataType)
	}
}

// Enhanced parseArrayType with better error handling
func parseArrayType(dataType string) (string, int, error) {
	dataType = strings.TrimSpace(dataType)

	// Check for empty data type
	if dataType == "" {
		return "", 0, fmt.Errorf("empty data type")
	}

	// Check if it's an array type
	if strings.Contains(dataType, "[") && strings.Contains(dataType, "]") {
		// Use regex to extract base type and array length
		re := regexp.MustCompile(`^(\w+)\[(\d+)\]$`)
		matches := re.FindStringSubmatch(dataType)
		if len(matches) != 3 {
			return "", 0, fmt.Errorf("invalid array type format: %s (expected format: type[count])", dataType)
		}

		baseType := matches[1]
		count, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", 0, fmt.Errorf("invalid array length in type %s: %w", dataType, err)
		}

		if count < 0 {
			return "", 0, fmt.Errorf("negative array length in type %s: %d", dataType, count)
		}

		return baseType, count, nil
	}

	// Not an array type, return base type with count 1
	return dataType, 1, nil
}

// Encode Bytes
func (r DeviceRegister) Encode() []byte {
	return r.Value[:]
}

// Decode Bytes
func (r *DeviceRegister) Decode(data []byte) {
	// Make sure r.Value has enough capacity
	if r.Value == nil || cap(r.Value) < len(data) {
		r.Value = make([]byte, len(data))
	} else {
		r.Value = r.Value[:len(data)]
	}
	copy(r.Value, data)
}

// To string
func (r DeviceRegister) String() string {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonData)
}

// CheckBit checks if a specific bit is set in a uint16 value
func CheckBit(num uint16, index uint16) bool {
	if index > 15 { // uint16 has 16 bits (0-15)
		return false
	}
	mask := uint16(1) << index
	return (num & mask) != 0
}

// reorderBytes reorders the bytes according to the specified byte order
func reorderBytes(data []byte, order string) []byte {
	length := len(data)

	switch order {
	case "A":
		if length >= 1 {
			return data[:1]
		}
	case "AB":
		if length >= 2 {
			return data[:2]
		}
	case "BA":
		if length >= 2 {
			return []byte{data[1], data[0]}
		}
	case "ABCD":
		if length >= 4 {
			return data[:4]
		}
	case "DCBA":
		if length >= 4 {
			return []byte{data[3], data[2], data[1], data[0]}
		}
	case "BADC":
		if length >= 4 {
			return []byte{data[1], data[0], data[3], data[2]}
		}
	case "CDAB":
		if length >= 4 {
			return []byte{data[2], data[3], data[0], data[1]}
		}
	case "ABCDEFGH":
		if length >= 8 {
			return data[:8]
		}
	case "HGFEDCBA":
		if length >= 8 {
			return []byte{data[7], data[6], data[5], data[4], data[3], data[2], data[1], data[0]}
		}
	case "BADCFEHG":
		if length >= 8 {
			return []byte{data[1], data[0], data[3], data[2], data[5], data[4], data[7], data[6]}
		}
	case "GHEFCDAB":
		if length >= 8 {
			return []byte{data[6], data[7], data[4], data[5], data[2], data[3], data[0], data[1]}
		}
	}

	// Default to returning the original data
	return data
}

// DecodedValue holds all possible interpretations of a raw Modbus value
type DecodedValue struct {
	Raw     []byte  `json:"raw"`     // Raw value as bytes
	Float64 float64 `json:"float64"` // Value as float64
	Type    string  `json:"type"`    // Type of the value
	AsType  any     `json:"asType"`  // Value as any type
}

// GetFloat64Value returns the Float64 value, optionally rounded to the specified number of decimal places
func (dv DecodedValue) GetFloat64Value(round int) float64 {
	if round > 0 {
		return math.Round(dv.Float64*math.Pow(10, float64(round))) / math.Pow(10, float64(round))
	}
	return dv.Float64
}

// ToString returns the string representation of the DecodedValue
func (dv DecodedValue) String() string {
	return fmt.Sprintf("Raw: % X, Float64: %f, AsType: % X", dv.Raw, dv.Float64, dv.AsType)
}

// float32FromBits converts a uint32 to a float32
func float32FromBits(bits uint32) float32 {
	return *(*float32)(unsafe.Pointer(&bits))
}

// float64FromBits converts a uint64 to a float64
func float64FromBits(bits uint64) float64 {
	return *(*float64)(unsafe.Pointer(&bits))
}

// FuzzyEqual compares two float64 values with a tolerance
func FuzzyEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
