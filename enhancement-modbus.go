package modbus

import "fmt"

func GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	sortRegisters(registers)

	var grouped [][]DeviceRegister
	for i := 0; i < len(registers); {
		currentGroup := []DeviceRegister{registers[i]}
		last := registers[i]
		i++
		for i < len(registers) {
			current := registers[i]
			if current.SlaverId == last.SlaverId && current.Function == last.Function && current.Address == last.Address+last.Quantity {
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

func sortRegisters(registers []DeviceRegister) {
	for i := 0; i < len(registers)-1; i++ {
		for j := 0; j < len(registers)-i-1; j++ {
			if registers[j].SlaverId > registers[j+1].SlaverId ||
				(registers[j].SlaverId == registers[j+1].SlaverId &&
					registers[j].Function > registers[j+1].Function) ||
				(registers[j].SlaverId == registers[j+1].SlaverId &&
					registers[j].Function == registers[j+1].Function &&
					registers[j].Address > registers[j+1].Address) {
				registers[j], registers[j+1] = registers[j+1], registers[j]
			}
		}
	}
}

func ReadGroupedData(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {
	var result [][]DeviceRegister
	for _, group := range grouped {
		start := group[0].Address
		var totalQuantity uint16
		for _, reg := range group {
			totalQuantity += reg.Quantity
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
				copy(group[i].Value[:group[i].Quantity*2], data[offset:offset+int(group[i].Quantity*2)])
				group[i].Status = "VALID:OK"
				offset += int(group[i].Quantity * 2)
			}
		}
		result = append(result, group)
	}
	return result
}