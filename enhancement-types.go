package modbus

import (
	"io"
)

// ModbusApi defines the interface for Modbus client operations.
type ModbusApi interface {
	// Handler API
	GetType() string     // GetType returns the type of the handler
	SetLogger(io.Writer) // SetLogger sets the logger for the client
	// basic methods
	ReadCoils(slaveID uint16, startAddress, quantity uint16) ([]bool, error)              // ReadCoils reads multiple coils
	ReadDiscreteInputs(slaveID uint16, startAddress, quantity uint16) ([]bool, error)     // ReadDiscreteInputs reads multiple discrete inputs
	ReadHoldingRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) // ReadHoldingRegisters reads multiple holding registers
	ReadInputRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error)   // ReadInputRegisters reads multiple input registers
	WriteSingleCoil(slaveID uint16, address uint16, value bool) error                     // WriteSingleCoil writes a single coil
	WriteSingleRegister(slaveID uint16, address, value uint16) error                      // WriteSingleRegister writes a single register
	WriteMultipleCoils(slaveID uint16, startAddress uint16, values []bool) error          // WriteMultipleCoils writes multiple coils
	WriteMultipleRegisters(slaveID uint16, startAddress uint16, values []uint16) error    // WriteMultipleRegisters writes multiple registers
	// Extended methods
	ReadCustomData(funcCode uint16, slaveID uint16, startAddress, quantity uint16) ([]byte, error) // ReadCustomData reads custom data
	WriteCustomData(funcCode uint16, slaveID uint16, startAddress uint16, data []byte) error       // WriteCustomData writes custom data
	ReadDeviceIdentity(slaveID uint16) (uint16, error)                                             // ReadDeviceIdentity reads device identity
	ReadExceptionStatus(slaveID uint16) (string, error)                                            // ReadExceptionStatus reads exception status
	// enhanced methods
	ReadWithMask(slaveID uint16, readAddress, andMask, orMask uint16) (uint16, error) // ReadWithMask reads a register and applies a mask
}
