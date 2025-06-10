package modbus

import (
	"fmt"
	"io"
)

const (
	// Bit access
	FuncCodeReadDiscreteInputs = 2
	FuncCodeReadCoils          = 1
	FuncCodeWriteSingleCoil    = 5
	FuncCodeWriteMultipleCoils = 15

	// 16-bit access
	FuncCodeReadInputRegisters         = 4
	FuncCodeReadHoldingRegisters       = 3
	FuncCodeWriteSingleRegister        = 6
	FuncCodeWriteMultipleRegisters     = 16
	FuncCodeReadWriteMultipleRegisters = 23
	FuncCodeMaskWriteRegister          = 22
	FuncCodeReadFIFOQueue              = 24
	//MEI
	FuncCodeMEI                     = 43
	MEITypeReadDeviceIdentification = 14
)

const (
	ExceptionCodeIllegalFunction                    = 1
	ExceptionCodeIllegalDataAddress                 = 2
	ExceptionCodeIllegalDataValue                   = 3
	ExceptionCodeServerDeviceFailure                = 4
	ExceptionCodeAcknowledge                        = 5
	ExceptionCodeServerDeviceBusy                   = 6
	ExceptionCodeMemoryParityError                  = 8
	ExceptionCodeGatewayPathUnavailable             = 10
	ExceptionCodeGatewayTargetDeviceFailedToRespond = 11
)

// Modbus TCP Protocol Identifier
const ProtocolIdentifierTCP uint16 = 0x0000

// Modbus Function Codes
const (
	FuncCodeReadExceptionStatus uint8 = 0x07
)

// ModbusError implements error interface.
type ModbusError struct {
	FunctionCode  byte
	ExceptionCode byte
}

// Error converts known modbus exception code to error message.
func (e *ModbusError) Error() string {
	var name string
	switch e.ExceptionCode {
	case ExceptionCodeIllegalFunction:
		name = "illegal function"
	case ExceptionCodeIllegalDataAddress:
		name = "illegal data address"
	case ExceptionCodeIllegalDataValue:
		name = "illegal data value"
	case ExceptionCodeServerDeviceFailure:
		name = "server device failure"
	case ExceptionCodeAcknowledge:
		name = "acknowledge"
	case ExceptionCodeServerDeviceBusy:
		name = "server device busy"
	case ExceptionCodeMemoryParityError:
		name = "memory parity error"
	case ExceptionCodeGatewayPathUnavailable:
		name = "gateway path unavailable"
	case ExceptionCodeGatewayTargetDeviceFailedToRespond:
		name = "gateway target device failed to respond"
	default:
		name = "unknown"
	}
	return fmt.Sprintf("modbus: exception '%v' (%s), function '%v'", e.ExceptionCode, name, e.FunctionCode)
}

// ModbusApi defines the interface for Modbus client operations.
type ModbusApi interface {
	// Handler API
	GetMode() string     // GetType returns the type of the handler: "tcp", "rtu", etc.
	SetLogger(io.Writer) // SetLogger sets the logger for the client
	// Standard methods
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
	ReadRawDeviceIdentity(slaveID uint16) ([]byte, error)                                          // ReadRawDeviceIdentity reads raw device identity data
	ReadRawData([]byte) ([]byte, error)                                                            // ReadRawData reads raw data
	ReadDeviceIdentityWithHandler(slaveID uint16, handler func([]byte) error) error                // ReadDeviceIdentityWithHandler reads device identity and processes it with a handler
	ReadWithMask(slaveID uint16, readAddress, andMask, orMask uint16) (uint16, error)              // ReadWithMask reads a register and applies a mask
}
