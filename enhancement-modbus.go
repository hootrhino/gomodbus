package modbus

import (
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"unsafe"
)

// GroupDeviceRegister groups consecutive DeviceRegisters by Function and SlaverId
func GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	sort.Slice(registers, func(i, j int) bool {
		if registers[i].SlaverId != registers[j].SlaverId {
			return registers[i].SlaverId < registers[j].SlaverId
		}
		if registers[i].Function != registers[j].Function {
			return registers[i].Function < registers[j].Function
		}
		return registers[i].Address < registers[j].Address
	})

	var groups [][]DeviceRegister
	if len(registers) == 0 {
		return groups
	}

	currentGroup := []DeviceRegister{registers[0]}
	last := registers[0]

	for i := 1; i < len(registers); i++ {
		r := registers[i]
		if r.Function == last.Function && r.SlaverId == last.SlaverId && r.Address <= last.Address+last.Quantity {
			currentGroup = append(currentGroup, r)
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []DeviceRegister{r}
		}
		last = r
	}
	groups = append(groups, currentGroup)

	return groups
}

// ReadData reads batches of grouped registers using the provided Modbus client
func ReadGroupedData(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {
	var result [][]DeviceRegister

	for _, group := range grouped {
		if len(group) == 0 {
			continue
		}
		start := group[0].Address
		end := start
		for _, reg := range group {
			if reg.Address+reg.Quantity > end {
				end = reg.Address + reg.Quantity
			}
		}
		quantity := end - start
		var data []byte
		var err error
		switch group[0].Function {
		case 1:
			data, err = client.ReadCoils(start, quantity)
		case 2:
			data, err = client.ReadDiscreteInputs(start, quantity)
		case 3:
			data, err = client.ReadHoldingRegisters(start, quantity)
		case 4:
			data, err = client.ReadInputRegisters(start, quantity)
		default:
			continue
		}
		if err != nil || len(data) < int(quantity*2) {
			continue
		}

		for i := range group {
			offset := (group[i].Address - start) * 2
			copy(group[i].Value[:], data[offset:offset+4]) // support max 4 bytes
		}
		result = append(result, group)
	}

	return result
}

// DecodedValue holds all possible interpretations of a raw Modbus value
type DecodedValue struct {
	Raw     []byte
	Float64 float64
	AsType  any
}

// DecodeValue decodes the raw value into float64 according to DataType and DataOrder
func DecodeValue(r DeviceRegister) float64 {
	val, _ := DecodeValueAsInterface(r)
	return val.Float64
}

// DecodeValueAsInterface returns the decoded result as multiple forms
func DecodeValueAsInterface(r DeviceRegister) (DecodedValue, error) {
	bytes := reorderBytes(r.Value, r.DataOrder)
	res := DecodedValue{Raw: bytes}

	switch r.DataType {
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
		v := float64FromBytes(bytes[:])
		res.AsType = v
		res.Float64 = v
	default:
		return res, errors.New("unsupported data type")
	}

	return res, nil
}

// reorderBytes reorders bytes according to DataOrder
func reorderBytes(data [4]byte, order string) []byte {
	switch order {
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

func float32FromBits(bits uint32) float32 {
	return *(*float32)(unsafe.Pointer(&bits))
}

func float64FromBytes(b []byte) float64 {
	var arr [8]byte
	copy(arr[:], b)
	return *(*float64)(unsafe.Pointer(&arr))
}

// FuzzyEqual compares two float64 values with a tolerance
func FuzzyEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
