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
	"sort"
	"testing"
)

func TestGroupDeviceRegisterWithLogicalContinuity(t *testing.T) {
	tests := []struct {
		name     string
		input    []DeviceRegister
		expected [][]DeviceRegister
	}{
		{
			name:     "Nil input",
			input:    nil,
			expected: [][]DeviceRegister{},
		},
		{
			name:     "Empty input",
			input:    []DeviceRegister{},
			expected: [][]DeviceRegister{},
		},
		{
			name: "Single register",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				},
			},
		},
		{
			name: "Two logically continuous registers, same SlaverId",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 1, ReadAddress: 110, ReadQuantity: 5},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
					{SlaverId: 1, ReadAddress: 110, ReadQuantity: 5},
				},
			},
		},
		{
			name: "Two non-continuous registers, same SlaverId",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 1, ReadAddress: 120, ReadQuantity: 5},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				},
				{
					{SlaverId: 1, ReadAddress: 120, ReadQuantity: 5},
				},
			},
		},
		{
			name: "Multiple registers with different SlaverIds",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 2, ReadAddress: 200, ReadQuantity: 5},
				{SlaverId: 1, ReadAddress: 110, ReadQuantity: 5},
				{SlaverId: 2, ReadAddress: 205, ReadQuantity: 3},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
					{SlaverId: 1, ReadAddress: 110, ReadQuantity: 5},
				},
				{
					{SlaverId: 2, ReadAddress: 200, ReadQuantity: 5},
					{SlaverId: 2, ReadAddress: 205, ReadQuantity: 3},
				},
			},
		},
		{
			name: "Multiple registers with mixed continuity",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 1, ReadAddress: 120, ReadQuantity: 5}, // Non-continuous with previous
				{SlaverId: 1, ReadAddress: 125, ReadQuantity: 5}, // Continuous with previous
				{SlaverId: 2, ReadAddress: 200, ReadQuantity: 5},
				{SlaverId: 2, ReadAddress: 210, ReadQuantity: 3}, // Non-continuous with previous
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				},
				{
					{SlaverId: 1, ReadAddress: 120, ReadQuantity: 5},
					{SlaverId: 1, ReadAddress: 125, ReadQuantity: 5},
				},
				{
					{SlaverId: 2, ReadAddress: 200, ReadQuantity: 5},
				},
				{
					{SlaverId: 2, ReadAddress: 210, ReadQuantity: 3},
				},
			},
		},
		{
			name: "Registers with zero ReadQuantity",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 0}, // Zero ReadQuantity
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 5},
				{SlaverId: 1, ReadAddress: 105, ReadQuantity: 5},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 0},
				},
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 5},
					{SlaverId: 1, ReadAddress: 105, ReadQuantity: 5},
				},
			},
		},
		{
			name: "Input with unsorted ReadAddresses",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 120, ReadQuantity: 10},
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 1, ReadAddress: 110, ReadQuantity: 10},
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
					{SlaverId: 1, ReadAddress: 110, ReadQuantity: 10},
					{SlaverId: 1, ReadAddress: 120, ReadQuantity: 10},
				},
			},
		},
		{
			name: "Register with max uint16 values",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 65530, ReadQuantity: 5},
				{SlaverId: 1, ReadAddress: 65535, ReadQuantity: 1}, // Max uint16 value
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 65530, ReadQuantity: 5},
					{SlaverId: 1, ReadAddress: 65535, ReadQuantity: 1},
				},
			},
		},
		{
			name: "Multiple SlaverIds with one empty after grouping",
			input: []DeviceRegister{
				{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				{SlaverId: 3, ReadAddress: 300, ReadQuantity: 5},
				// SlaverId 2 is missing
			},
			expected: [][]DeviceRegister{
				{
					{SlaverId: 1, ReadAddress: 100, ReadQuantity: 10},
				},
				{
					{SlaverId: 3, ReadAddress: 300, ReadQuantity: 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupDeviceRegisterWithLogicalContinuity(tt.input)
			printDeviceRegisters(t, got)
			sortGroups(got)
			sortGroups(tt.expected)
			// Compare the actual result with expected result
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GroupDeviceRegisterWithLogicalContinuity() = %v, want %v", got, tt.expected)
				// Print detailed comparison for debugging
				t.Logf("Got %d groups, expected %d groups", len(got), len(tt.expected))
				for i := range max(len(got), len(tt.expected)) {
					if i < len(got) && i < len(tt.expected) {
						t.Logf("Group %d: got %v, expected %v", i, got[i], tt.expected[i])
					} else if i < len(got) {
						t.Logf("Group %d: got %v, expected none", i, got[i])
					} else {
						t.Logf("Group %d: got none, expected %v", i, tt.expected[i])
					}
				}
			}
		})
	}
}

// Helper function to sort groups of DeviceRegister by ReadAddress
func sortGroups(groups [][]DeviceRegister) {
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool {
			return group[i].ReadAddress < group[j].ReadAddress
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i]) == 0 || len(groups[j]) == 0 {
			return len(groups[i]) < len(groups[j])
		}
		return groups[i][0].ReadAddress < groups[j][0].ReadAddress
	})
}

// Helper function for test debugging
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// beautiful print [][]DeviceRegister
func printDeviceRegisters(t *testing.T, groups [][]DeviceRegister) {
	for _, group := range groups {
		t.Log("Group:")
		for _, reg := range group {
			t.Logf("  SlaverId: %d, ReadAddress: %d, ReadQuantity: %d\n", reg.SlaverId, reg.ReadAddress, reg.ReadQuantity)
		}
	}
}

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
