package modbus

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"unsafe"
)

// DeviceRegister represents a Modbus register with metadata
type DeviceRegister struct {
	Tag       string  `json:"tag"`       // A unique identifier or label for the register
	Alias     string  `json:"alias"`     // A human-readable name or alias for the register
	Function  int     `json:"function"`  // Modbus function code (e.g., 3 for Read Holding Registers)
	SlaverId  byte    `json:"slaverId"`  // ID of the Modbus slave device
	Address   uint16  `json:"address"`   // Address of the register in the Modbus device
	Frequency int64   `json:"frequency"` // Polling frequency in milliseconds
	Quantity  uint16  `json:"quantity"`  // Number of registers to read/write
	DataType  string  `json:"dataType"`  // Data type of the register value (e.g., uint16, int32, float32)
	BitMask   uint16  `json:"bitMask"`   // Bitmask for bit-level operations (e.g., 0x01, 0x02)
	DataOrder string  `json:"dataOrder"` // Byte order for multi-byte values (e.g., ABCD, DCBA)
	Weight    float64 `json:"weight"`    // Scaling factor for the register value
	Value     [8]byte `json:"value"`     // Raw value of the register as a byte array
}

// Encode Bytes
func (r DeviceRegister) Encode() []byte {
	return r.Value[:]
}

// Decode Bytes
func (r *DeviceRegister) Decode(data []byte) {
	copy(r.Value[:], data)
}

// To string
func (r DeviceRegister) String() string {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonData)
}

func CheckBit(num uint16, index uint16) bool {
	if index < 0 || index > 15 {
		return false
	}
	mask := uint16(1) << index
	return (num & mask) != 0
}

// DecodeValueAsInterface returns the decoded result as multiple forms
func (r DeviceRegister) DecodeValue() (DecodedValue, error) {
	bytes := reorderBytes(r.Value, r.DataOrder)
	res := DecodedValue{Raw: bytes}

	switch r.DataType {
	case "bool":
		Uint16 := binary.BigEndian.Uint16(bytes[:2])
		res.AsType = CheckBit(Uint16, r.BitMask)
		res.Float64 = 0
		if res.AsType.(bool) {
			res.Float64 = 1
		}
	case "byte":
		v := bytes[0]
		res.AsType = v
		res.Float64 = float64(v)
	case "uint8":
		v := uint8(bytes[0])
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "int8":
		v := int8(bytes[0])
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "uint16":
		v := binary.BigEndian.Uint16(bytes[:2])
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "int16":
		v := int16(binary.BigEndian.Uint16(bytes[:2]))
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "uint32":
		v := binary.BigEndian.Uint32(bytes[:4])
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "int32":
		v := int32(binary.BigEndian.Uint32(bytes[:4]))
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "float32":
		bits := binary.BigEndian.Uint32(bytes[:4])
		v := float32FromBits(bits)
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	case "float64":
		// Ensure we have enough bytes for float64
		bits := binary.BigEndian.Uint64(bytes[:8])
		v := float64FromBits(bits)
		res.AsType = v
		res.Float64 = float64(v) * r.Weight
	default:
		return res, errors.New("unsupported data type:" + r.DataType)
	}

	return res, nil
}

// reorderBytes reorders bytes according to DataOrder
func reorderBytes(data [8]byte, order string) []byte {
	switch order {
	case "A":
		return data[:1]
	case "AB":
		return data[:2]
	case "BA":
		return []byte{data[1], data[0]}
	case "ABCD":
		return data[:]
	case "DCBA":
		return []byte{data[3], data[2], data[1], data[0]}
	case "BADC":
		return []byte{data[1], data[0], data[3], data[2]}
	case "CDAB":
		return []byte{data[2], data[3], data[0], data[1]}
	default:
		return data[:]
	}
}

// DecodedValue holds all possible interpretations of a raw Modbus value
type DecodedValue struct {
	Raw     []byte  `json:"raw"`     // Raw value as bytes
	Float64 float64 `json:"float64"` // Value as float64
	AsType  any     `json:"asType"`  // Value as any type
}

// ToString returns the string representation of the DecodedValue
func (dv DecodedValue) String() string {
	return fmt.Sprintf("Raw: %v, Float64: %f, AsType: %v", dv.Raw, dv.Float64, dv.AsType)
}

func float32FromBits(bits uint32) float32 {
	return *(*float32)(unsafe.Pointer(&bits))
}
func float64FromBits(bits uint64) float64 {
	return *(*float64)(unsafe.Pointer(&bits))
}

// FuzzyEqual compares two float64 values with a tolerance
func FuzzyEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
