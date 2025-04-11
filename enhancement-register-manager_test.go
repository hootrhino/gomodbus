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
	defer client.GetTransporter().Close()
	manager := NewRegisterManager(client, "UART", 10)
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		t.Log(registers)
	})
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.Start()
	reg1 := DeviceRegister{}
	reg1.Tag = "Tag-1"
	reg1.Alias = "Alias-1"
	reg1.SlaverId = 1
	reg1.Function = 3
	reg1.Address = 1
	reg1.Quantity = 1
	reg1.DataType = "uint16"
	reg1.DataOrder = "AB"
	reg1.Frequency = 10
	reg1.Weight = 1
	reg1.Value = [8]byte{0}
	//
	reg2 := DeviceRegister{}
	reg2.Tag = "Tag-1"
	reg2.Alias = "Alias-1"
	reg2.SlaverId = 1
	reg2.Function = 3
	reg2.Address = 1
	reg2.Quantity = 1
	reg2.DataType = "uint16"
	reg2.DataOrder = "AB"
	reg2.Frequency = 10
	reg2.Weight = 1
	reg2.Value = [8]byte{0}
	registers := []DeviceRegister{
		reg1,
		reg2,
	}
	manager.LoadRegisters(registers)
	manager.SetOnErrorCallback(func(err error) {
		t.Log(err)
	})
	manager.SetOnReadCallback(func(registers []DeviceRegister) {
		t.Log(registers)
	})
	for i := 0; i < 100; i++ {
		manager.ReadGroupedData()
	}
	time.Sleep(1 * time.Second)
	manager.Stop()
}
