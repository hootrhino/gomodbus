package modbus

import (
	"fmt"
	"sort"
	"sync"
)

// GroupDeviceRegisterWithLogicalContinuity groups registers based on logical continuity.
// Logical continuity is determined by checking if the sum of the ReadAddress and ReadQuantity
// of one register matches the ReadAddress of the next register.
func GroupDeviceRegisterWithLogicalContinuity(registers []DeviceRegister) [][]DeviceRegister {
	// Early return for empty input
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	// Step 1: Group registers by SlaverId
	slaverGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range registers {
		slaverGroups[reg.SlaverId] = append(slaverGroups[reg.SlaverId], reg)
	}

	// Final result container
	var result [][]DeviceRegister

	// Step 2: Process each SlaverId group
	for _, regs := range slaverGroups {
		// Sort registers by ReadAddress
		sort.Slice(regs, func(i, j int) bool {
			return regs[i].ReadAddress < regs[j].ReadAddress
		})

		// Step 3: Split into logically continuous groups
		var currentGroup []DeviceRegister
		currentGroup = append(currentGroup, regs[0])

		for i := 1; i < len(regs); i++ {
			// Check if the current register is logically continuous with the previous one
			if regs[i].ReadAddress == regs[i-1].ReadAddress+regs[i-1].ReadQuantity {
				// If logically continuous, add to the current group
				currentGroup = append(currentGroup, regs[i])
			} else {
				// If not logically continuous, finalize the current group and start a new one
				result = append(result, currentGroup)
				currentGroup = []DeviceRegister{regs[i]}
			}
		}

		// Add the last group
		if len(currentGroup) > 0 {
			result = append(result, currentGroup)
		}
	}

	return result
}

func readGroup(client Client, group []DeviceRegister) ([]DeviceRegister, error) {
	client.SetSlaveId(group[0].SlaverId)
	start := group[0].ReadAddress
	var totalQuantity uint16
	for _, reg := range group {
		totalQuantity += reg.ReadQuantity
	}
	var data []byte
	var err error
	switch group[0].Function {
	case 1:
		data, err = client.ReadCoils(start, totalQuantity)
	case 2:
		data, err = client.ReadDiscreteInputs(start, totalQuantity)
	case 3:
		data, err = client.ReadHoldingRegisters(start, totalQuantity)
	case 4:
		data, err = client.ReadInputRegisters(start, totalQuantity)
	default:
		return nil, fmt.Errorf("unsupported Modbus function code: %d", group[0].Function)
	}
	if err != nil {
		for i := range group {
			group[i].Status = fmt.Sprintf("INVALID:%s", err)
		}
		return group, err
	}
	offset := 0
	for i := range group {
		expectedLength := int(group[i].ReadQuantity * 2)
		if offset+expectedLength > len(data) {
			msg := fmt.Sprintf("Error: Data out of bounds for register %d (SlaverId=%d, ReadAddress=%d, offset=%d, expected=%d, dataLength=%d)",
				i, group[i].SlaverId, group[i].ReadAddress, offset, expectedLength, len(data))
			group[i].Status = msg
			break
		}

		// Ensure the Value slice has enough capacity
		if cap(group[i].Value) < expectedLength {
			group[i].Value = make([]byte, expectedLength)
		} else {
			group[i].Value = group[i].Value[:expectedLength]
		}

		// Copy data safely
		copy(group[i].Value, data[offset:offset+expectedLength])
		group[i].Status = "VALID:OK"
		offset += expectedLength
	}
	return group, nil
}

// Read data from modbus server concurrently
func ReadGroupedDataConcurrently(client Client, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
	var wg sync.WaitGroup
	var result [][]DeviceRegister
	var mu sync.Mutex
	errors := []error{}

	for _, group := range grouped {
		wg.Add(1)
		go func(group []DeviceRegister) {
			mu.Lock()
			defer wg.Done()
			defer mu.Unlock()
			groupResult, err := readGroup(client, group)
			if err != nil {
				errors = append(errors, err)
			}
			result = append(result, groupResult)
		}(group)
	}
	wg.Wait()
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
