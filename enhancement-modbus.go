package modbus

import (
	"sort"
)

// GroupDeviceRegister groups consecutive DeviceRegisters by Function and SlaverId
func GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	sort.Slice(registers, func(i, j int) bool {
		if registers[i].SlaverId != registers[j].SlaverId {
			return registers[i].SlaverId < registers[j].SlaverId
		}
		if registers[i].Function != registers[j].Function {
			return registers[i].Function < registers[j].Function
		}
		return registers[i].Address < registers[j].Address
	})

	var groups [][]DeviceRegister
	if len(registers) == 0 {
		return groups
	}

	currentGroup := []DeviceRegister{registers[0]}
	last := registers[0]

	for i := 1; i < len(registers); i++ {
		r := registers[i]
		if r.Function == last.Function && r.SlaverId == last.SlaverId && r.Address <= last.Address+last.Quantity {
			currentGroup = append(currentGroup, r)
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []DeviceRegister{r}
		}
		last = r
	}
	groups = append(groups, currentGroup)

	return groups
}

// ReadData reads batches of grouped registers using the provided Modbus client
func ReadGroupedData(client Client, grouped [][]DeviceRegister) [][]DeviceRegister {
	var result [][]DeviceRegister

	for _, group := range grouped {
		if len(group) == 0 {
			continue
		}
		start := group[0].Address
		end := start
		for _, reg := range group {
			if reg.Address+reg.Quantity > end {
				end = reg.Address + reg.Quantity
			}
		}
		quantity := end - start
		var data []byte
		var err error
		switch group[0].Function {
		case 1:
			data, err = client.ReadCoils(start, quantity)
		case 2:
			data, err = client.ReadDiscreteInputs(start, quantity)
		case 3:
			data, err = client.ReadHoldingRegisters(start, quantity)
		case 4:
			data, err = client.ReadInputRegisters(start, quantity)
		default:
			continue
		}
		if err != nil || len(data) < int(quantity*2) {
			continue
		}

		for i := range group {
			offset := (group[i].Address - start) * 2
			copy(group[i].Value[:], data[offset:offset+4]) // support max 4 bytes
		}
		result = append(result, group)
	}

	return result
}
