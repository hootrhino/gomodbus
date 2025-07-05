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
	DataType     string  `json:"dataType"`     // Data type of the register value (e.g., uint16, int32, float32, string)
	DataOrder    string  `json:"dataOrder"`    // Byte order for multi-byte values (e.g., ABCD, DCBA)
	BitPosition  uint16  `json:"bitPosition"`  // Bit position for bit-level operations (e.g., 0, 1, 2)
	BitMask      uint16  `json:"bitMask"`      // Bitmask for bit-level operations (e.g., 0x01, 0x02)
	Weight       float64 `json:"weight"`       // Scaling factor for the register value
	Frequency    uint64  `json:"frequency"`    // Polling frequency in milliseconds
	Value        []byte  `json:"value"`        // Raw value of the register as a byte array (variable length)
	Status       string  `json:"status"`       // Status of the register (e.g., "OK", "Error")
}

// DecodeValue converts the raw bytes in the register to a typed value based on the DataType
func (r DeviceRegister) DecodeValue() (DecodedValue, error) {
	requiredBytes, err := getRequiredBytes(r.DataType)
	if err != nil {
		return DecodedValue{Raw: r.Value}, err
	}
	// For non-string, check length before reorder
	if r.DataType != "string" && len(r.Value) < requiredBytes {
		return DecodedValue{Raw: r.Value}, fmt.Errorf("not enough bytes for data type %s: have %d, need %d",
			r.DataType, len(r.Value), requiredBytes)
	}
	bytes := reorderBytes(r.Value, r.DataOrder)
	res := DecodedValue{Raw: bytes, Type: r.DataType}

	switch r.DataType {
	case "bitfield":
		if len(bytes) < 2 {
			return res, fmt.Errorf("not enough bytes for bitfield: need at least 2")
		}
		val := binary.BigEndian.Uint16(bytes[:2]) & r.BitMask
		res.AsType = val
		res.Float64 = float64(val) * r.Weight
	case "bool":
		if len(bytes) < 2 {
			return res, fmt.Errorf("not enough bytes for bool: need at least 2")
		}
		val := binary.BigEndian.Uint16(bytes[:2])
		b := CheckBit(val, r.BitPosition)
		res.AsType = b
		res.Float64 = 1.0
		if !b {
			res.Float64 = 0.0
		}
	case "byte", "uint8":
		if len(bytes) < 1 {
			return res, fmt.Errorf("not enough bytes for %s: need at least 1", r.DataType)
		}
		res.AsType = bytes[0]
		res.Float64 = float64(bytes[0]) * r.Weight
	case "int8":
		if len(bytes) < 1 {
			return res, fmt.Errorf("not enough bytes for int8: need at least 1")
		}
		res.AsType = int8(bytes[0])
		res.Float64 = float64(res.AsType.(int8)) * r.Weight
	case "uint16":
		val := binary.BigEndian.Uint16(bytes[:2])
		res.AsType = val
		res.Float64 = float64(val) * r.Weight
	case "int16":
		val := int16(binary.BigEndian.Uint16(bytes[:2]))
		res.AsType = val
		res.Float64 = float64(val) * r.Weight
	case "uint32":
		val := binary.BigEndian.Uint32(bytes[:4])
		res.AsType = val
		res.Float64 = float64(val) * r.Weight
	case "int32":
		val := int32(binary.BigEndian.Uint32(bytes[:4]))
		res.AsType = val
		res.Float64 = float64(val) * r.Weight
	case "float32":
		bits := binary.BigEndian.Uint32(bytes[:4])
		v := float32FromBits(bits)
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "float64":
		bits := binary.BigEndian.Uint64(bytes[:8])
		v := float64FromBits(bits)
		res.AsType = v
		res.Float64 = v * r.Weight
	case "string":
		res.AsType = string(bytes)
		res.Float64 = 0
	default:
		return res, fmt.Errorf("unsupported data type: %s", r.DataType)
	}
	return res, nil
}

// getRequiredBytes returns the number of bytes required for a given data type
func getRequiredBytes(dataType string) (int, error) {
	switch dataType {
	case "bitfield", "bool", "uint16", "int16":
		return 2, nil
	case "uint32", "int32", "float32":
		return 4, nil
	case "uint64", "float64":
		return 8, nil
	case "byte", "uint8", "int8":
		return 1, nil
	case "string":
		// String type can have variable length
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown data type: %s", dataType)
	}
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
	return fmt.Sprintf("Raw: %v, Float64: %f, AsType: %v", dv.Raw, dv.Float64, dv.AsType)
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
