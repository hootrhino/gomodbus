package modbus

import (
	"os"
	"testing"
	"time"
)

func Test_RegisterManager(t *testing.T) {
	handler := NewRTUClientHandler("COM3")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)

	err := handler.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	client := NewClient(handler)
	defer client.Close()
	manager := NewRegisterManager(client, 10)

	registers := []DeviceRegister{
		{
			Tag:       "Tag-0-bool",
			Alias:     "Alias-0-bool",
			SlaverId:  1,
			Function:  3,
			Address:   0,
			Quantity:  1,
			DataType:  "bool",
			BitMask:   0x0001,
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
		{
			Tag:       "Tag-1",
			Alias:     "Alias-1",
			SlaverId:  1,
			Function:  3,
			Address:   1,
			Quantity:  1,
			DataType:  "uint16",
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
		{
			Tag:       "Tag-2",
			Alias:     "Alias-2",
			SlaverId:  1,
			Function:  3,
			Address:   2,
			Quantity:  1,
			DataType:  "uint16",
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
		// no continued registers
		{
			Tag:       "Tag-10-bool",
			Alias:     "Alias-10-bool",
			SlaverId:  1,
			Function:  3,
			Address:   10,
			Quantity:  1,
			DataType:  "bool",
			BitMask:   0x0001,
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
		{
			Tag:       "Tag-11",
			Alias:     "Alias-11",
			SlaverId:  1,
			Function:  3,
			Address:   11,
			Quantity:  1,
			DataType:  "uint16",
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
		{
			Tag:       "Tag-12",
			Alias:     "Alias-12",
			SlaverId:  1,
			Function:  3,
			Address:   12,
			Quantity:  1,
			DataType:  "uint16",
			DataOrder: "AB",
			Frequency: 10,
			Weight:    1,
			Value:     [8]byte{0},
		},
	}
	manager.LoadRegisters(registers)
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Log(err)
			}
			t.Log("== ", register.Tag, register.Alias, register.DataType,
				register.DataOrder, register.BitMask, register.Weight, register.Frequency, value)
		}
	})
	manager.Start()
	for range 100 {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}
