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
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Import the actual function from your package
// For testing purposes, we're including it here directly
func GroupDeviceRegisterWithLogicalContinuity(registers []DeviceRegister) [][]DeviceRegister {
	// Early return for empty or nil input
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	// Step 1: Group registers by SlaverId
	slaverGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range registers {
		// Create a copy of the register to avoid potential side effects
		regCopy := reg
		slaverGroups[reg.SlaverId] = append(slaverGroups[reg.SlaverId], regCopy)
	}

	// Final result container
	result := make([][]DeviceRegister, 0, len(slaverGroups)) // Pre-allocate with estimated capacity

	// Step 2: Process each SlaverId group
	for _, regs := range slaverGroups {
		// Skip empty slaver groups
		if len(regs) == 0 {
			continue
		}

		// Sort registers by ReadAddress
		sortByReadAddress(regs)

		// Step 3: Split into logically continuous groups
		currentGroup := make([]DeviceRegister, 0, min(len(regs), 8)) // Pre-allocate with reasonable capacity
		currentGroup = append(currentGroup, regs[0])

		for i := 1; i < len(regs); i++ {
			// Check for ReadQuantity being zero which could cause logic issues
			if regs[i-1].ReadQuantity == 0 {
				// Finalize current group and start a new one to avoid infinite loops or logic errors
				result = append(result, currentGroup)
				currentGroup = make([]DeviceRegister, 0, min(len(regs)-i, 8))
				currentGroup = append(currentGroup, regs[i])
				continue
			}

			// Check if the current register is logically continuous with the previous one
			if regs[i].ReadAddress == regs[i-1].ReadAddress+regs[i-1].ReadQuantity {
				// If logically continuous, add to the current group
				currentGroup = append(currentGroup, regs[i])
			} else {
				// If not logically continuous, finalize the current group and start a new one
				result = append(result, currentGroup)
				currentGroup = make([]DeviceRegister, 0, min(len(regs)-i, 8))
				currentGroup = append(currentGroup, regs[i])
			}
		}

		// Add the last group
		if len(currentGroup) > 0 {
			result = append(result, currentGroup)
		}
	}

	return result
}

// Helper function for Go versions < 1.21 which don't have built-in min for ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 1. Use sort.Slice instead of bubble sort for better performance
func sortByReadAddress(regs []DeviceRegister) {
	sort.Slice(regs, func(i, j int) bool {
		return regs[i].ReadAddress < regs[j].ReadAddress
	})
}

// readGroup reads a group of DeviceRegister from the Modbus device
func readGroup(client ModbusApi, group []DeviceRegister) ([]DeviceRegister, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("cannot read empty group")
	}

	slaveID := uint16(group[0].SlaverId)
	start := group[0].ReadAddress
	var totalQuantity uint16
	for _, reg := range group {
		totalQuantity += reg.ReadQuantity
	}

	var data interface{}
	var err error

	switch group[0].Function {
	case 1:
		data, err = client.ReadCoils(slaveID, start, totalQuantity)
	case 2:
		data, err = client.ReadDiscreteInputs(slaveID, start, totalQuantity)
	case 3, 4:
		if group[0].Function == 3 {
			data, err = client.ReadHoldingRegisters(slaveID, start, totalQuantity)
		} else {
			data, err = client.ReadInputRegisters(slaveID, start, totalQuantity)
		}
	default:
		return nil, fmt.Errorf("unsupported Modbus function code: %d", group[0].Function)
	}

	if err != nil {
		return handleReadError(group, err)
	}

	return parseAndUpdateGroup(group, data)
}

// handleReadError handles the error when reading data from the Modbus device
func handleReadError(group []DeviceRegister, err error) ([]DeviceRegister, error) {
	for i := range group {
		group[i].Status = fmt.Sprintf("INVALID:%s", err)
	}
	return group, fmt.Errorf("modbus read error (slave %d, addr %d): %w", group[0].SlaverId, group[0].ReadAddress, err)
}

// parseAndUpdateGroup parses the data and updates the status of the group
func parseAndUpdateGroup(group []DeviceRegister, data interface{}) ([]DeviceRegister, error) {
	offset := 0
	for i := range group {
		qty := int(group[i].ReadQuantity)
		var err error
		switch group[0].Function {
		case 1, 2:
			boolData, ok := data.([]bool)
			if !ok {
				return nil, errors.New("invalid data type for coils or discrete inputs")
			}
			err = parseBoolData(group[i], boolData, offset, qty)
		case 3, 4:
			uint16Data, ok := data.([]uint16)
			if !ok {
				return nil, errors.New("invalid data type for holding or input registers")
			}
			err = parseUint16Data(group[i], uint16Data, offset, qty)
		}
		if err != nil {
			return group, err
		}
		offset += qty
	}
	return group, nil
}

// parseBoolData parses the boolean data and updates the register value and status
func parseBoolData(reg DeviceRegister, data []bool, offset, qty int) error {
	if offset+qty > len(data) {
		msg := fmt.Sprintf("Data out of bounds for register (SlaverId=%d, ReadAddress=%d, offset=%d, qty=%d, dataLen=%d)",
			reg.SlaverId, reg.ReadAddress, offset, qty, len(data))
		reg.Status = "INVALID:" + msg
		return errors.New(msg)
	}
	reg.Value = make([]byte, qty)
	for j := 0; j < qty; j++ {
		if data[offset+j] {
			reg.Value[j] = 1
		} else {
			reg.Value[j] = 0
		}
	}
	reg.Status = "VALID:OK"
	return nil
}

// parseUint16Data parses the uint16 data and updates the register value and status
func parseUint16Data(reg DeviceRegister, data []uint16, offset, qty int) error {
	if offset+qty > len(data) {
		msg := fmt.Sprintf("Register data out of bounds for register (SlaverId=%d, ReadAddress=%d, offset=%d, qty=%d, dataLen=%d)",
			reg.SlaverId, reg.ReadAddress, offset, qty, len(data))
		reg.Status = "INVALID:" + msg
		return errors.New(msg)
	}
	reg.Value = make([]byte, qty*2)
	for j := 0; j < qty; j++ {
		reg.Value[j*2] = byte(data[offset+j] >> 8)
		reg.Value[j*2+1] = byte(data[offset+j])
	}
	reg.Status = "VALID:OK"
	return nil
}

// 3. Add context to error handling in concurrent reader
func ReadGroupedDataConcurrently(client ModbusApi, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
	var wg sync.WaitGroup
	result := make([][]DeviceRegister, len(grouped))

	// Use errgroup for better error handling
	type groupError struct {
		groupIndex int
		err        error
	}

	errChan := make(chan groupError, len(grouped))

	for i, group := range grouped {
		wg.Add(1)
		go func(idx int, group []DeviceRegister) {
			defer wg.Done()

			groupResult, err := readGroup(client, group)
			result[idx] = groupResult

			if err != nil {
				errChan <- groupError{idx, err}
			}
		}(i, group)
	}

	// Use a separate goroutine to collect errors
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	errors := make([]error, 0)
	for ge := range errChan {
		errors = append(errors, fmt.Errorf("group %d error: %w", ge.groupIndex, ge.err))
	}

	return result, errors
}

// Read data from modbus server sequentially
func ReadGroupedDataSequential(client ModbusApi, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
	var result [][]DeviceRegister
	var errors []error
	for _, group := range grouped {
		groupResult, err := readGroup(client, group)
		if err != nil {
			errors = append(errors, err)
		}
		result = append(result, groupResult)
	}
	return result, errors
}
