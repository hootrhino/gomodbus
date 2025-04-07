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
	BitMask   uint8   `json:"bitMask"`   // Bitmask for bit-level operations (e.g., 0x01, 0x02)
	DataOrder string  `json:"dataOrder"` // Byte order for multi-byte values (e.g., ABCD, DCBA)
	Weight    float64 `json:"weight"`    // Scaling factor for the register value
	Value     [4]byte `json:"value"`     // Raw value of the register as a byte array
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

// DecodeValueAsInterface returns the decoded result as multiple forms
func (r DeviceRegister) DecodeValue() (DecodedValue, error) {

	bytes := reorderBytes(r.Value, r.DataOrder)
	res := DecodedValue{Raw: bytes}

	switch r.DataType {
	case "bitfield":
		if len(bytes) < 1 {
			return res, errors.New("not enough bytes for bitfield")
		}
		if r.BitMask == 0 {
			return res, errors.New("bitMask is not set")
		}
		v := bytes[0] & r.BitMask
		res.AsType = v //bytes[0] is uint8
		res.Float64 = float64(v)
	case "uint8":
		v := uint8(bytes[0])
		res.AsType = v
		res.Float64 = float64(v)
	case "int8":
		v := int8(bytes[0])
		res.AsType = v
		res.Float64 = float64(v)
	case "uint16":
		v := binary.BigEndian.Uint16(bytes[:2])
		res.AsType = v
		res.Float64 = float64(v)
	case "int16":
		v := int16(binary.BigEndian.Uint16(bytes[:2]))
		res.AsType = v
		res.Float64 = float64(v)
	case "uint32":
		v := binary.BigEndian.Uint32(bytes[:4])
		res.AsType = v
		res.Float64 = float64(v)
	case "int32":
		v := int32(binary.BigEndian.Uint32(bytes[:4]))
		res.AsType = v
		res.Float64 = float64(v)
	case "float32":
		bits := binary.BigEndian.Uint32(bytes[:4])
		v := float32FromBits(bits)
		res.AsType = v
		res.Float64 = float64(v)
	case "float64":
		if len(bytes) < 8 {
			return res, errors.New("not enough bytes for float64")
		}
		// Ensure we have enough bytes for float64
		bits := binary.BigEndian.Uint64(bytes[:8])
		v := float64FromBits(bits)
		res.AsType = v
		res.Float64 = float64(v)
	default:
		return res, errors.New("unsupported data type")
	}

	return res, nil
}

// reorderBytes reorders bytes according to DataOrder
func reorderBytes(data [4]byte, order string) []byte {
	switch order {
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
	Raw     []byte  `json:"raw"`
	Float64 float64 `json:"float64"`
	AsType  any     `json:"asType"`
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
