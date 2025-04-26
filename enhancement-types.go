package modbus


// ModbusApi defines the interface for Modbus client operations.
type ModbusApi interface {
	// Read Coils
	ReadCoils(slaveID uint16, startAddress, quantity uint16) ([]bool, error)
	// Read Discrete Inputs
	ReadDiscreteInputs(slaveID uint16, startAddress, quantity uint16) ([]bool, error)
	// Read Holding Registers
	ReadHoldingRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error)
	// Read Input Registers
	ReadInputRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error)
	// Write Single Coil
	WriteSingleCoil(slaveID uint16, address uint16, value bool) error
	// Write Single Register
	WriteSingleRegister(slaveID uint16, address, value uint16) error
	// Write Multiple Coils
	WriteMultipleCoils(slaveID uint16, startAddress uint16, values []bool) error
	// Write Multiple Registers
	WriteMultipleRegisters(slaveID uint16, startAddress uint16, values []uint16) error
	// Read Custom Data
	ReadCustomData(funcCode uint16, slaveID uint16, startAddress, quantity uint16) ([]byte, error)
	// Write Custom Data
	WriteCustomData(funcCode uint16, slaveID uint16, startAddress uint16, data []byte) error
	// Read Device Identity
	ReadDeviceIdentity(slaveID uint16) (string, error)
	// Read Exception Status
	ReadExceptionStatus(slaveID uint16) (string, error)
	// Data type specific reads
	ReadUint8(slaveID uint16, address uint16, byteOrder string) (uint8, error)
	ReadUint16(slaveID uint16, address uint16, byteOrder string) (uint16, error)
	ReadUint32(slaveID uint16, address uint16, byteOrder string) (uint32, error)
	ReadUint64(slaveID uint16, address uint16, byteOrder string) (uint64, error)
	ReadInt8(slaveID uint16, address uint16, byteOrder string) (int8, error)
	ReadInt16(slaveID uint16, address uint16, byteOrder string) (int16, error)
	ReadInt32(slaveID uint16, address uint16, byteOrder string) (int32, error)
	ReadInt64(slaveID uint16, address uint16, byteOrder string) (int64, error)
	ReadFloat32(slaveID uint16, address uint16, byteOrder string) (float32, error)
	ReadFloat64(slaveID uint16, address uint16, byteOrder string) (float64, error)
	ReadBool(slaveID uint16, address uint16, byteOrder string) (bool, error)
	ReadBytes(slaveID uint16, address uint16, length uint16, byteOrder string) ([]byte, error)
	ReadBit(slaveID uint16, address uint16, bit uint8, byteOrder string) (bool, error)
	ReadBits(slaveID uint16, address uint16, startBit uint8, length uint8, byteOrder string) ([]bool, error)
	ReadString(slaveID uint16, address uint16, length uint16, encoding string, byteOrder string) (string, error)
}
