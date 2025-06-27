package modbus

import "testing"

// assertUint8Equal checks if two slices of uint8 are equal.
func assertUint16Equal(t *testing.T, expected []uint16, actual []uint16) {
	if len(expected) != len(actual) {
		t.Errorf("Expected length %d, but got %d", len(expected), len(actual))
		return
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("Expected %v, but got %v", expected, actual)
		}
	}
}
