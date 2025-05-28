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

// 2. Enhanced error handling in readGroup to provide more context
func readGroup(client Client, group []DeviceRegister) ([]DeviceRegister, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("cannot read empty group")
	}

	client.SetSlaveId(group[0].SlaverId)
	start := group[0].ReadAddress
	var totalQuantity uint16
	for _, reg := range group {
		totalQuantity += reg.ReadQuantity
	}

	var data []byte
	var err error

	// Function code validation before attempting to read
	validFunctions := map[uint8]bool{1: true, 2: true, 3: true, 4: true}
	if !validFunctions[group[0].Function] {
		return nil, fmt.Errorf("unsupported Modbus function code: %d", group[0].Function)
	}

	switch group[0].Function {
	case 1:
		data, err = client.ReadCoils(start, totalQuantity)
	case 2:
		data, err = client.ReadDiscreteInputs(start, totalQuantity)
	case 3:
		data, err = client.ReadHoldingRegisters(start, totalQuantity)
	case 4:
		data, err = client.ReadInputRegisters(start, totalQuantity)
	}

	if err != nil {
		for i := range group {
			group[i].Status = fmt.Sprintf("INVALID:%s", err)
		}
		return group, fmt.Errorf("modbus read error (slave %d, addr %d): %w",
			group[0].SlaverId, start, err)
	}

	// Process data into individual registers
	offset := 0
	for i := range group {
		expectedLength := int(group[i].ReadQuantity * 2)
		if offset+expectedLength > len(data) {
			msg := fmt.Sprintf("Data out of bounds for register %d (SlaverId=%d, ReadAddress=%d, offset=%d, expected=%d, dataLength=%d)",
				i, group[i].SlaverId, group[i].ReadAddress, offset, expectedLength, len(data))
			group[i].Status = "INVALID:" + msg
			return group, errors.New(msg)
		}
		// Handle virtual registers, Value can be set by application to FFFF for virtual registers
		if group[i].Type == RegisterTypeVirtual {
			group[i].Value = []byte{0xFF, 0xFF}
		} else {
			// Ensure the Value slice has enough capacity
			if cap(group[i].Value) < expectedLength {
				group[i].Value = make([]byte, expectedLength)
			} else {
				group[i].Value = group[i].Value[:expectedLength]
			}

		}
		// Copy data safely
		copy(group[i].Value, data[offset:offset+expectedLength])
		group[i].Status = "VALID:OK"
		offset += expectedLength
	}

	return group, nil
}

// 3. Add context to error handling in concurrent reader
func ReadGroupedDataConcurrently(client Client, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
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
func ReadGroupedDataSequential(client Client, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
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
