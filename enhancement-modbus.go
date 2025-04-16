package modbus

import (
	"fmt"
	"sort"
	"sync"
)

// GroupDeviceRegister groups registers by SlaveId and consecutive ReadAddress
// Returns a slice of register groups, where each group contains registers with
// the same SlaveId and consecutive ReadAddress values
// If multiple registers have identical SlaverId, ReadAddress, and ReadQuantity,
// they will be combined into a single entry
func GroupDeviceRegisterWithUniqueSlaverId(registers []DeviceRegister) [][]DeviceRegister {
	// Early return for empty input
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	// Step 1: Combine identical registers (same SlaverId, ReadAddress, ReadQuantity)
	type registerKey struct {
		SlaverId     uint8
		ReadAddress  uint16
		ReadQuantity uint16
	}

	uniqueRegisters := make(map[registerKey]DeviceRegister)
	for _, reg := range registers {
		key := registerKey{
			SlaverId:     reg.SlaverId,
			ReadAddress:  reg.ReadAddress,
			ReadQuantity: reg.ReadQuantity,
		}

		// If this is the first time we're seeing this key, or we want to replace
		// with this register for some reason, store it
		if _, exists := uniqueRegisters[key]; !exists {
			uniqueRegisters[key] = reg
		}
		// Note: Here you could implement logic to merge metadata or choose which
		// register to keep if you have specific rules for combining identical registers
	}

	// Convert unique registers back to slice
	deduplicatedRegisters := make([]DeviceRegister, 0, len(uniqueRegisters))
	for _, reg := range uniqueRegisters {
		deduplicatedRegisters = append(deduplicatedRegisters, reg)
	}

	// Step 2: Group registers by SlaveId
	slaverGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range deduplicatedRegisters {
		if reg.DataType == "bool" {
		}
		slaverGroups[reg.SlaverId] = append(slaverGroups[reg.SlaverId], reg)
	}

	// Final result container
	var result [][]DeviceRegister

	// Step 3: Process each SlaveId group
	for _, regs := range slaverGroups {
		// Sort registers by ReadAddress to identify consecutive addresses
		sort.Slice(regs, func(i, j int) bool {
			return regs[i].ReadAddress < regs[j].ReadAddress
		})

		// Step 4: Split each SlaveId group into subgroups of consecutive addresses
		var currentGroup []DeviceRegister
		currentGroup = append(currentGroup, regs[0])

		for i := 1; i < len(regs); i++ {
			// Check if current register's address is consecutive to the previous one
			if regs[i].ReadAddress == regs[i-1].ReadAddress+regs[i-1].ReadQuantity {
				// If consecutive, add to current group
				currentGroup = append(currentGroup, regs[i])
			} else {
				// If not consecutive, finalize current group and start a new one
				result = append(result, currentGroup)
				currentGroup = []DeviceRegister{regs[i]}
			}
		}

		// Don't forget to add the last group
		if len(currentGroup) > 0 {
			result = append(result, currentGroup)
		}
	}

	return result
}

// GroupDeviceRegister groups registers by SlaveId and consecutive ReadAddress
// Returns a slice of register groups, where each group contains registers with
// the same SlaveId and consecutive ReadAddress values
// This version does not deduplicate registers with identical SlaverId
func GroupDeviceRegisterWithUniqueAddress(registers []DeviceRegister) [][]DeviceRegister {
	// Early return for empty input
	if len(registers) == 0 {
		return [][]DeviceRegister{}
	}

	// Step 1: Group registers by SlaveId
	slaveGroups := make(map[uint8][]DeviceRegister)
	for _, reg := range registers {
		slaveGroups[reg.SlaverId] = append(slaveGroups[reg.SlaverId], reg)
	}

	// Final result container
	var result [][]DeviceRegister

	// Step 2: Process each SlaveId group
	for _, regs := range slaveGroups {
		// Sort registers by ReadAddress to identify consecutive addresses
		sort.Slice(regs, func(i, j int) bool {
			return regs[i].ReadAddress < regs[j].ReadAddress
		})

		// Step 3: Split each SlaveId group into subgroups of consecutive addresses
		// Also ensure function codes match within each group
		if len(regs) > 0 {
			var currentGroup []DeviceRegister
			currentGroup = append(currentGroup, regs[0])

			for i := 1; i < len(regs); i++ {
				// Check if current register's address is consecutive to the previous one
				// and that they have the same function code
				if regs[i].ReadAddress == regs[i-1].ReadAddress+regs[i-1].ReadQuantity &&
					regs[i].Function == regs[i-1].Function {
					// If consecutive and same function, add to current group
					currentGroup = append(currentGroup, regs[i])
				} else {
					// If not consecutive or different function, finalize current group and start a new one
					result = append(result, currentGroup)
					currentGroup = []DeviceRegister{regs[i]}
				}
			}

			// Don't forget to add the last group
			if len(currentGroup) > 0 {
				result = append(result, currentGroup)
			}
		}
	}

	return result
}

// Read data from modbus server concurrently
func ReadGroupedDataConcurrently(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {
	var wg sync.WaitGroup
	var result [][]DeviceRegister
	var mu sync.Mutex
	for _, group := range grouped {
		wg.Add(1)
		go func(group []DeviceRegister) {
			defer wg.Done()
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
				fmt.Printf("Unsupported Modbus function code: %d\n", group[0].Function)
				return

			}
			if err != nil {
				fmt.Printf("Error reading registers: %v\n", err)
			}
			mu.Lock()
			defer mu.Unlock()
			offset := 0
			for i := range group {
				copy(group[i].Value[:group[i].ReadQuantity*2], data[offset:offset+int(group[i].ReadQuantity*2)])
				if err != nil {
					group[i].Status = fmt.Sprintf("INVALID:%s", err)
				} else {
					group[i].Status = "VALID:OK"
				}
				offset += int(group[i].ReadQuantity * 2)
			}
			result = append(result, group)
		}(group)
	}
	wg.Wait()
	return result
}

// Read data from modbus server sequentially
func ReadGroupedDataSequential(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {

	var result [][]DeviceRegister
	for _, group := range grouped {
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
			fmt.Printf("Unsupported Modbus function code: %d\n", group[0].Function)
			continue
		}
		if err != nil {
			fmt.Printf("Error reading registers: %v\n", err)
			for i := range group {
				group[i].Status = fmt.Sprintf("INVALID:%s", err)
			}
		} else {
			offset := 0
			for i := range group {
				copy(group[i].Value[:group[i].ReadQuantity*2], data[offset:offset+int(group[i].ReadQuantity*2)])
				group[i].Status = "VALID:OK"
				offset += int(group[i].ReadQuantity * 2)
			}
		}
		result = append(result, group)
	}
	return result
}
