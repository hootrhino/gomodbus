package modbus

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func MakeNewTestClient() Client {
	handler := NewRTUClientHandler("COM3")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)
	err := handler.Connect()
	if err != nil {
		panic(err)
	}
	client := NewClient(handler)
	return client
}
func Test_RegisterManager_Decode_bool(t *testing.T) {
	client := MakeNewTestClient()
	defer client.Close()
	manager := NewRegisterManager(client, 10)

	registers := []DeviceRegister{}
	for i := range 16 {
		registers = append(registers, DeviceRegister{
			Tag:          fmt.Sprintf("tag:bool:%d", i),
			Alias:        fmt.Sprintf("tag:bool:%d", i),
			SlaverId:     uint8(i),
			Function:     3,
			ReadAddress:  1,
			ReadQuantity: 1,
			DataType:     "bool", // ABFF = 10101011 11111111
			BitPosition:  uint16(i),
		})
	}
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(fmt.Sprintf("tag:%s, value:%v", register.Tag, value.AsType))
		}
	})
	if errLoad := manager.LoadRegisters(registers); errLoad != nil {
		t.Fatal(errLoad)
	}

	manager.Start()
	for range 1 {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}
func Test_RegisterManager_Decode_uint16(t *testing.T) {
	client := MakeNewTestClient()
	defer client.Close()
	manager := NewRegisterManager(client, 10)

	registers := []DeviceRegister{}

	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint16-1:44031:%d", 1),
		Alias:        fmt.Sprintf("tag:uint16-2:44031:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 1,
		DataType:     "uint16", // ABFF = 10101011 11111111 = 44031
		DataOrder:    "AB",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint16-2:44031:%d", 1),
		Alias:        fmt.Sprintf("tag:uint16-2:44031:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 1,
		DataType:     "uint16", // ABFF = 10101011 11111111 = 44031
		DataOrder:    "AB",
	})
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(fmt.Sprintf("tag:%s, value:%v", register.Tag, value.AsType))
		}
	})
	if errLoad := manager.LoadRegisters(registers); errLoad != nil {
		t.Fatal(errLoad)
	}

	manager.Start()
	for range 1 {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}

func Test_RegisterManager_Decode_uint32(t *testing.T) {
	client := MakeNewTestClient()
	defer client.Close()
	manager := NewRegisterManager(client, 10)
	registers := []DeviceRegister{}

	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint32-1:1078530010:%d", 1),
		Alias:        fmt.Sprintf("tag:uint32-1:1078530010:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 2,
		DataType:     "uint32", // 40 49 0f da = 01000000010010010000111111011010 = 1078530010
		DataOrder:    "ABCD",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:uint32-2:1078530010:%d", 1),
		Alias:        fmt.Sprintf("tag:uint32-2:1078530010:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 2,
		DataType:     "uint32", // 40 49 0f da = 01000000010010010000111111011010 = 1078530010
		DataOrder:    "ABCD",
	})
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(fmt.Sprintf("tag:%s, value:%v", register.Tag, value.AsType))
		}
	})
	if errLoad := manager.LoadRegisters(registers); errLoad != nil {
		t.Fatal(errLoad)
	}

	manager.Start()
	for range 1 {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}

func Test_RegisterManager_Decode_float32(t *testing.T) {
	client := MakeNewTestClient()
	defer client.Close()
	manager := NewRegisterManager(client, 10)
	registers := []DeviceRegister{}
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:float3232-1:3.1415926:%d", 1),
		Alias:        fmt.Sprintf("tag:float3232-1:3.1415926:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 2,
		DataType:     "float32", // 40 49 0f da = 01000000010010010000111111011010 = 3.1415926
		DataOrder:    "ABCD",
	})
	registers = append(registers, DeviceRegister{
		Tag:          fmt.Sprintf("tag:float3232-2:3.1415926:%d", 1),
		Alias:        fmt.Sprintf("tag:float3232-2:3.1415926:%d", 1),
		SlaverId:     uint8(1),
		Function:     3,
		ReadAddress:  0,
		ReadQuantity: 2,
		DataType:     "float32", // 40 49 0f da = 01000000010010010000111111011010 = 3.1415926
		DataOrder:    "ABCD",
	})
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		for _, register := range registers {
			value, err := register.DecodeValue()
			if err != nil {
				t.Fatal(err)
			}
			t.Log(fmt.Sprintf("tag:%s, value:%v", register.Tag, value.AsType))
		}
	})
	if errLoad := manager.LoadRegisters(registers); errLoad != nil {
		t.Fatal(errLoad)
	}
	manager.Start()
	for range 1 {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}
