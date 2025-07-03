// transporter.go (or similar new file)
package modbus

// ModbusTransporter defines the common interface for different Modbus communication modes.
type ModbusTransporter interface {
	Send(slaveID uint8, pdu []byte) error
	Receive() (slaveID uint8, pdu []byte, err error)
	// Add other common methods like Close() if all transporters should implement them
	RemoteAddr() string // For logging, if applicable to all
}
