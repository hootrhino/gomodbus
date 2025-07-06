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
	registers := []DeviceRegister{
		{SlaverId: 1, ReadAddress: 300, ReadQuantity: 10},
		{SlaverId: 1, ReadAddress: 100, ReadQuantity: 5},
		{SlaverId: 1, ReadAddress: 200, ReadQuantity: 15},
	}

	expected := []DeviceRegister{
		{SlaverId: 1, ReadAddress: 100, ReadQuantity: 5},
		{SlaverId: 1, ReadAddress: 200, ReadQuantity: 15},
		{SlaverId: 1, ReadAddress: 300, ReadQuantity: 10},
	}

	sortByReadAddress(registers)

	if !reflect.DeepEqual(registers, expected) {
		t.Errorf("sortByReadAddress() = %v, want %v", registers, expected)
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
