// Copyright (C) 2024  wwhai
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <https://www.gnu.org/licenses/>.

package modbus

import (
	"io"
)

// ModbusApi defines the interface for Modbus client operations.
type ModbusApi interface {
	// Handler API
	GetLastModbusError() *ModbusError // GetLastError returns the last Modbus error encountered by this handler
	GetMode() string                  // GetType returns the type of the handler: "tcp", "rtu", etc.
	SetLogger(io.Writer)              // SetLogger sets the logger for the client
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
	ReadRawData([]byte) ([]byte, error)                                                            // ReadRawData reads raw data
}
