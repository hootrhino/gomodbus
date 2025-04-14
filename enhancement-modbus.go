package modbus

import (
	"fmt"
	"sync"
)

// sort registers
func GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	{ // inlined sortRegisters function
		for i := range len(registers) - 1 {
			for j := range len(registers) - i - 1 {
				if registers[j].SlaverId > registers[j+1].SlaverId ||
					(registers[j].SlaverId == registers[j+1].SlaverId &&
						registers[j].Function > registers[j+1].Function) ||
					(registers[j].SlaverId == registers[j+1].SlaverId &&
						registers[j].Function == registers[j+1].Function &&
						registers[j].ReadAddress > registers[j+1].ReadAddress) {
					registers[j], registers[j+1] = registers[j+1], registers[j]
				}
			}
		}
	}
	var grouped [][]DeviceRegister
	for i := 0; i < len(registers); {
		currentGroup := []DeviceRegister{registers[i]}
		last := registers[i]
		i++
		for i < len(registers) {
			current := registers[i]
			if current.SlaverId == last.SlaverId &&
				current.Function == last.Function &&
				current.ReadAddress == last.ReadAddress+last.ReadAddress {
				currentGroup = append(currentGroup, current)
				last = current
				i++
			} else {
				break
			}
		}
		grouped = append(grouped, currentGroup)
	}
	return grouped
}

//	func sortRegisters(registers []DeviceRegister) {
//		for i := 0; i < len(registers)-1; i++ {
//			for j := 0; j < len(registers)-i-1; j++ {
//				if registers[j].SlaverId > registers[j+1].SlaverId ||
//					(registers[j].SlaverId == registers[j+1].SlaverId &&
//						registers[j].Function > registers[j+1].Function) ||
//					(registers[j].SlaverId == registers[j+1].SlaverId &&
//						registers[j].Function == registers[j+1].Function &&
//						registers[j].Address > registers[j+1].Address) {
//					registers[j], registers[j+1] = registers[j+1], registers[j]
//				}
//			}
//		}
//	}

// Read data from modbus server concurrently
func ReadGroupedDataConcurrently(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {
	var wg sync.WaitGroup
	var result [][]DeviceRegister
	var mu sync.Mutex
	for _, group := range grouped {
		wg.Add(1)
		go func(group []DeviceRegister) {
			defer wg.Done()
			start := group[0].ReadAddress
			var totalQuantity uint16
			for _, reg := range group {
				totalQuantity += reg.ReadAddress
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
				group[i].Status = "VALID:OK"
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
