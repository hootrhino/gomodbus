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

func GroupDeviceRegisterWithLogicalContinuity(registers []DeviceRegister) [][]DeviceRegister {
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	regsCopy := make([]DeviceRegister, len(registers))
	copy(regsCopy, registers)

	for i := range regsCopy {
		if regsCopy[i].ReadQuantity == 0 {
			if err := regsCopy[i].CalculateReadQuantity(); err != nil {
				continue
			}
		}
	}

	slaverGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range regsCopy {
		slaverGroups[reg.SlaverId] = append(slaverGroups[reg.SlaverId], reg)
	}

	result := make([][]DeviceRegister, 0, len(slaverGroups))

	for _, regs := range slaverGroups {
		if len(regs) == 0 {
			continue
		}

		sort.Slice(regs, func(i, j int) bool {
			return regs[i].ReadAddress < regs[j].ReadAddress
		})

		currentGroup := []DeviceRegister{regs[0]}

		for i := 1; i < len(regs); i++ {
			prev := regs[i-1]
			curr := regs[i]

			if prev.ReadQuantity == 0 {
				result = append(result, currentGroup)
				currentGroup = []DeviceRegister{curr}
				continue
			}

			if curr.ReadAddress == prev.ReadAddress+prev.ReadQuantity {
				currentGroup = append(currentGroup, curr)
			} else {
				result = append(result, currentGroup)
				currentGroup = []DeviceRegister{curr}
			}
		}

		if len(currentGroup) > 0 {
			result = append(result, currentGroup)
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sortByReadAddress(regs []DeviceRegister) {
	sort.SliceStable(regs, func(i, j int) bool {
		return regs[i].ReadAddress < regs[j].ReadAddress
	})
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

	var data interface{}
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

func parseAndUpdateGroup(group []DeviceRegister, data interface{}) ([]DeviceRegister, error) {
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
