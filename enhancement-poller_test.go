package modbus

import (
	"sync/atomic"
	"testing"
	"time"

	goserial "github.com/hootrhino/goserial"
)

func TestModbusDevicePollerWithRTU(t *testing.T) {
	port, err := goserial.Open(&goserial.Config{
		Address:  "COM3",
		BaudRate: 9600,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  5000 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()

	handler := NewModbusRTUHandler(port, 1*time.Second)

	tests := []struct {
		name            string
		registers       []DeviceRegister
		expectedCalls   int
		expectedDataLen int
	}{
		{
			name: "RTU Poller with Success",
			registers: []DeviceRegister{
				{Tag: "reg1", SlaverId: 1, ReadAddress: 0, ReadQuantity: 5, Function: 3}, // 假设读取保持寄存器
			},
			expectedCalls:   1,
			expectedDataLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewModbusRegisterManager(handler, 10)
			if err := mgr.LoadRegisters(tt.registers); err != nil {
				t.Fatalf("LoadRegisters failed: %v", err)
			}

			poller := NewModbusDevicePoller(100 * time.Millisecond)
			poller.AddManager(mgr)

			var dataReceived int32
			var errorReceived int32
			mgr.SetOnData(func(data []DeviceRegister) {
				atomic.AddInt32(&dataReceived, 1)
				if len(data) != tt.expectedDataLen {
					t.Errorf("expected %d registers, got %d", tt.expectedDataLen, len(data))
				}
				for _, reg := range data {
					if len(reg.Value) == 0 {
						t.Errorf("register %s has empty value", reg.Tag)
					}
				}
			})

			mgr.SetOnError(func(err error) {
				atomic.AddInt32(&errorReceived, 1)
				t.Errorf("unexpected error: %v", err)
			})

			poller.Start()
			defer poller.Stop()

			time.Sleep(250 * time.Millisecond)

			if atomic.LoadInt32(&dataReceived) == 0 {
				t.Error("expected data callback to be called, but it wasn't")
			}
			if atomic.LoadInt32(&errorReceived) > 0 {
				t.Errorf("expected no errors, but got %d", errorReceived)
			}
		})
	}
}
