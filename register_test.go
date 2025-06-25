package modbus

import (
	"encoding/binary"
	"math"
	"testing"
)

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
