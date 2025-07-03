// transporter.go (or similar new file)
package modbus

// ModbusTransporter defines the common interface for different Modbus communication modes.
type ModbusTransporter interface {
	Send(slaveID uint8, pdu []byte) error
	Receive() (slaveID uint8, pdu []byte, err error)
	RemoteAddr() string
}
