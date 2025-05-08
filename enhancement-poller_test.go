package modbus

import (
	"fmt"
	"testing"
	"time"
)

func Test_Modbus_RegisterManager(t *testing.T) {
	client := MakeNewTestUartClient()
	defer client.Close()
	manager := NewModbusRegisterManager(client, 10)

	registers := []DeviceRegister{}

	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint16-1:0xABCD:%d", 1),
		Alias:        fmt.Sprintf("tag:uint16-1:0xABCD:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 1,
		DataType:     "uint16", // 0xABCD
		DataOrder:    "AB",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint16-2:0xABCD:%d", 1),
		Alias:        fmt.Sprintf("tag:uint16-2:0xABCD:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  1,
		ReadQuantity: 1,
		DataType:     "uint16", // 0xABCD
		DataOrder:    "AB",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint32-1:0xABCD:%d", 1),
		Alias:        fmt.Sprintf("tag:uint32-1:0xABCD:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  100,
		ReadQuantity: 2,
		DataType:     "uint32", // 0xABCD
		DataOrder:    "ABCD",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint32-2:0xABCD:%d", 1),
		Alias:        fmt.Sprintf("tag:uint32-2:0xABCD:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  102,
		ReadQuantity: 2,
		DataType:     "uint32", // 0xABCD
		DataOrder:    "ABCD",
	})

	if errLoad := manager.LoadRegisters(registers); errLoad != nil {
		t.Fatal(errLoad)
	}
	manager.Stream.SetOnData(func(data []DeviceRegister) {
		for _, r := range data {
			fmt.Printf("TAG: %s, Addr: %04X, Val: %d\n", r.Tag, r.ReadAddress, r.Value)
		}
	})

	manager.Stream.SetOnError(func(err error) {
		t.Logf("error during read: %v", err)
	})
	manager.Stream.Start()
	errs := manager.ReadAndStream()
	if len(errs) > 0 {
		for _, err := range errs {
			t.Log("read error:", err)
		}
	}
	defer manager.Stream.Stop()
	time.Sleep(2 * time.Second)
}
