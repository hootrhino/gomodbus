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

// GroupDeviceRegisterWithLogicalContinuity groups device registers by slave ID and logical continuity
// with support for array data types. It automatically calculates ReadQuantity for registers
// that don't have it set, including proper handling of array types like uint16[10], float32[5], etc.
func GroupDeviceRegisterWithLogicalContinuity(registers []DeviceRegister) [][]DeviceRegister {
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	// Create a copy to avoid modifying the original slice
	regsCopy := make([]DeviceRegister, len(registers))
	copy(regsCopy, registers)

	// Calculate ReadQuantity for registers that need it, including array types
	for i := range regsCopy {
		if regsCopy[i].ReadQuantity == 0 {
			if readQuantity, err := regsCopy[i].CalculateReadQuantity(); err != nil {
				// Log error but continue processing other registers
				fmt.Printf("Warning: Failed to calculate ReadQuantity for register %s: %v\n", regsCopy[i].Tag, err)
				continue
			} else {
				regsCopy[i].ReadQuantity = readQuantity
			}
		}
	}

	// Group registers by slave ID first
	slaverGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range regsCopy {
		// Skip registers with invalid ReadQuantity
		if reg.ReadQuantity == 0 {
			fmt.Printf("Warning: Skipping register %s with ReadQuantity=0\n", reg.Tag)
			continue
		}
		slaverGroups[reg.SlaverId] = append(slaverGroups[reg.SlaverId], reg)
	}

	result := make([][]DeviceRegister, 0, len(slaverGroups))

	// Process each slave group
	for _, regs := range slaverGroups {
		if len(regs) == 0 {
			continue
		}

		// Sort registers by function code first, then by read address
		sort.Slice(regs, func(i, j int) bool {
			if regs[i].Function != regs[j].Function {
				return regs[i].Function < regs[j].Function
			}
			return regs[i].ReadAddress < regs[j].ReadAddress
		})

		// Group by function code and logical continuity
		functionGroups := make(map[uint8][]DeviceRegister)
		for _, reg := range regs {
			functionGroups[reg.Function] = append(functionGroups[reg.Function], reg)
		}

		// Process each function group separately
		for _, funcRegs := range functionGroups {
			if len(funcRegs) == 0 {
				continue
			}

			// Sort by read address within function group
			sort.Slice(funcRegs, func(i, j int) bool {
				return funcRegs[i].ReadAddress < funcRegs[j].ReadAddress
			})

			// Group by logical continuity
			currentGroup := []DeviceRegister{funcRegs[0]}

			for i := 1; i < len(funcRegs); i++ {
				prev := funcRegs[i-1]
				curr := funcRegs[i]

				// Check if current register is logically continuous with previous
				expectedNextAddress := prev.ReadAddress + prev.ReadQuantity

				if curr.ReadAddress == expectedNextAddress {
					// Check if adding this register would exceed Modbus limits
					if canAddToGroup(currentGroup, curr) {
						currentGroup = append(currentGroup, curr)
					} else {
						// Start new group if limits would be exceeded
						result = append(result, currentGroup)
						currentGroup = []DeviceRegister{curr}
					}
				} else {
					// Address gap found, start new group
					result = append(result, currentGroup)
					currentGroup = []DeviceRegister{curr}
				}
			}

			// Add the last group
			if len(currentGroup) > 0 {
				result = append(result, currentGroup)
			}
		}
	}

	// Sort result groups for consistent output
	sort.Slice(result, func(i, j int) bool {
		if len(result[i]) == 0 || len(result[j]) == 0 {
			return len(result[i]) > len(result[j])
		}

		groupI := result[i][0]
		groupJ := result[j][0]

		if groupI.SlaverId != groupJ.SlaverId {
			return groupI.SlaverId < groupJ.SlaverId
		}
		if groupI.Function != groupJ.Function {
			return groupI.Function < groupJ.Function
		}
		return groupI.ReadAddress < groupJ.ReadAddress
	})

	return result
}

// canAddToGroup checks if adding a register to a group would exceed Modbus protocol limits
func canAddToGroup(group []DeviceRegister, newReg DeviceRegister) bool {
	if len(group) == 0 {
		return true
	}

	// Calculate total quantity if we add the new register
	totalQuantity := uint16(0)
	for _, reg := range group {
		totalQuantity += reg.ReadQuantity
	}
	totalQuantity += newReg.ReadQuantity

	// Check Modbus protocol limits based on function code
	switch newReg.Function {
	case 1, 2: // Read Coils, Read Discrete Inputs
		// Maximum 2000 bits per request
		return totalQuantity <= 2000
	case 3, 4: // Read Holding Registers, Read Input Registers
		// Maximum 125 registers per request
		return totalQuantity <= 125
	default:
		// Conservative limit for unknown function codes
		return totalQuantity <= 125
	}
}

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

	var data any
	var err error
	switch group[0].Function {
	case 1:
		data, err = client.ReadCoils(slaveID, start, totalQuantity)
	case 2:
		data, err = client.ReadDiscreteInputs(slaveID, start, totalQuantity)
	case 3:
		data, err = client.ReadHoldingRegisters(slaveID, start, totalQuantity)
	case 4:
		data, err = client.ReadInputRegisters(slaveID, start, totalQuantity)
	default:
		return nil, fmt.Errorf("unsupported Modbus function code: %d", group[0].Function)
	}
	if err != nil {
		return handleReadError(group, err)
	}
	return parseAndUpdateGroup(group, data)
}

func handleReadError(group []DeviceRegister, err error) ([]DeviceRegister, error) {
	for i := range group {
		group[i].Status = fmt.Sprintf("INVALID:%s", err)
	}
	return group, fmt.Errorf("modbus read error (slave %d, addr %d): %w", group[0].SlaverId, group[0].ReadAddress, err)
}

func parseAndUpdateGroup(group []DeviceRegister, data any) ([]DeviceRegister, error) {
	offset := 0
	for i := range group {
		reg := &group[i]
		qty := int(reg.ReadQuantity)
		var err error
		switch reg.Function {
		case 1, 2:
			boolData, ok := data.([]bool)
			if !ok {
				return nil, errors.New("invalid data type for coils or discrete inputs")
			}
			err = parseBoolData(reg, boolData, offset, qty)
		case 3, 4:
			uint16Data, ok := data.([]uint16)
			if !ok {
				return nil, errors.New("invalid data type for holding or input registers")
			}
			err = parseUint16Data(reg, uint16Data, offset, qty)
		}
		if err != nil {
			return group, err
		}
		offset += qty
	}
	return group, nil
}

func parseBoolData(reg *DeviceRegister, data []bool, offset, qty int) error {
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

func parseUint16Data(reg *DeviceRegister, data []uint16, offset, qty int) error {
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

func ReadGroupedDataConcurrently(client ModbusApi, grouped [][]DeviceRegister) ([][]DeviceRegister, []error) {
	var wg sync.WaitGroup
	result := make([][]DeviceRegister, len(grouped))
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
	go func() {
		wg.Wait()
		close(errChan)
	}()

	errors := make([]error, 0)
	for ge := range errChan {
		errors = append(errors, fmt.Errorf("group %d error: %w", ge.groupIndex, ge.err))
	}
	return result, errors
}

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
