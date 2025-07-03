package modbus

import "fmt"

// assertUint8Equal checks if two slices of uint8 are equal.
func AssertUint16Equal(expected []uint16, actual []uint16) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("expected length %d, but got %d", len(expected), len(actual))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return fmt.Errorf("expected %v, but got %v", expected, actual)
		}
	}
	return nil
}
