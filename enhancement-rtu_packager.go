package modbus

import (
	"fmt"
)

// RTUPackager handles Modbus RTU packet packing and unpacking.
type RTUPackager struct{}

// NewRTUPackager creates a new RTUPackager.
func NewRTUPackager() *RTUPackager {
	return &RTUPackager{}
}

// Pack packs a Modbus RTU PDU into a complete RTU frame.
// The RTU frame format is: Slave Address (1 byte) + Function Code (1 byte) + Data (variable length) + CRC (2 bytes).
func (p *RTUPackager) Pack(slaveID uint8, pdu []byte) ([]byte, error) {
	frame := make([]byte, 1+len(pdu)+2)
	frame[0] = slaveID
	copy(frame[1:], pdu)
	crc := CRC16(frame[:len(frame)-2])
	frame[len(frame)-1] = byte(crc & 0xFF)
	frame[len(frame)-2] = byte(crc >> 8)
	return frame, nil
}

// Unpack unpacks a Modbus RTU frame into a Slave Address and PDU. It also verifies the CRC.
func (p *RTUPackager) Unpack(frame []byte) (slaveID uint8, pdu []byte, err error) {
	if len(frame) < 3 { // Minimum length: Slave Address (1) + Function Code (1) + CRC (2) - but CRC needs data
		err = fmt.Errorf("invalid RTU frame length: %d bytes", len(frame))
		return
	}

	// Need at least 3 bytes for slave ID, function code and at least one data byte for CRC to be valid
	if len(frame) < 3 {
		err = fmt.Errorf("invalid RTU frame length: %d, minimum is 3", len(frame))
		return
	}

	receivedCRC := uint16(frame[len(frame)-1])<<8 | uint16(frame[len(frame)-2])
	calculatedCRC := CRC16(frame[:len(frame)-2])

	if receivedCRC != calculatedCRC {
		err = fmt.Errorf("CRC mismatch: received 0x%04X, calculated 0x%04X", receivedCRC, calculatedCRC)
		return
	}

	slaveID = frame[0]
	pdu = frame[1 : len(frame)-2]
	return
}