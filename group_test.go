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
	"reflect"
	"testing"
)

// TestSortByReadAddress tests the sorting helper function
func TestSortByReadAddress(t *testing.T) {
	// Example registers with array types
	registers := []DeviceRegister{
		{Tag: "temp_array", SlaverId: 1, Function: 3, ReadAddress: 100, DataType: "float32[5]"},
		{Tag: "status_bits", SlaverId: 1, Function: 3, ReadAddress: 110, DataType: "uint16[3]"},
		{Tag: "single_val", SlaverId: 1, Function: 3, ReadAddress: 113, DataType: "int32"},
	}

	groups := GroupDeviceRegisterWithLogicalContinuity(registers)

	// Test if groups are correctly sorted by SlaverId and ReadAddress
	for i := 0; i < len(groups)-1; i++ {
		if groups[i][0].SlaverId > groups[i+1][0].SlaverId ||
			(groups[i][0].SlaverId == groups[i+1][0].SlaverId && groups[i][0].ReadAddress > groups[i+1][0].ReadAddress) {
			t.Errorf("Groups not sorted as expected at index %d", i)
		}
	}
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{0, 5, 0},
		{-5, 5, -5},
		{5, 5, 5},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestEdgeCases tests additional edge cases
func TestEdgeCases(t *testing.T) {
	// Test with single SlaverId but multiple non-continuous groups
	registers := []DeviceRegister{
		{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
		{SlaverId: 1, ReadAddress: 200, ReadQuantity: 5},
		{SlaverId: 1, ReadAddress: 205, ReadQuantity: 5},
		{SlaverId: 1, ReadAddress: 300, ReadQuantity: 15},
	}

	expected := [][]DeviceRegister{
		{
			{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
		},
		{
			{SlaverId: 1, ReadAddress: 200, ReadQuantity: 5},
			{SlaverId: 1, ReadAddress: 205, ReadQuantity: 5},
		},
		{
			{SlaverId: 1, ReadAddress: 300, ReadQuantity: 15},
		},
	}

	result := GroupDeviceRegisterWithLogicalContinuity(registers)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Edge case test failed: got %v, want %v", result, expected)
	}
}
