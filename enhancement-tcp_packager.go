package modbus

import (
	"encoding/binary"
	"fmt"
)

// TCPPackager handles Modbus TCP packet packing and unpacking.
type TCPPackager struct{}

// NewTCPPackager creates a new TCPPackager.
func NewTCPPackager() *TCPPackager {
	return &TCPPackager{}
}

// Pack packs a Modbus TCP PDU into a complete TCP frame.
// The TCP frame format is: MBAP (7 bytes) + PDU (variable length).
// MBAP format: Transaction Identifier (2 bytes) + Protocol Identifier (2 bytes) + Length (2 bytes) + Unit Identifier (1 byte).
func (p *TCPPackager) Pack(transactionID uint16, unitID uint8, pdu []byte) ([]byte, error) {
	length := uint16(len(pdu) + 1) // Length field includes the Unit Identifier and PDU
	frame := make([]byte, 7+len(pdu))

	binary.BigEndian.PutUint16(frame[0:2], transactionID)
	binary.BigEndian.PutUint16(frame[2:4], ProtocolIdentifierTCP)
	binary.BigEndian.PutUint16(frame[4:6], length)
	frame[6] = unitID
	copy(frame[7:], pdu)

	return frame, nil
}

// Unpack unpacks a Modbus TCP frame into a Transaction Identifier, Unit Identifier, and PDU.
func (p *TCPPackager) Unpack(frame []byte) (transactionID uint16, unitID uint8, pdu []byte, err error) {
	if len(frame) < 7 {
		err = fmt.Errorf("invalid TCP frame length: %d bytes", len(frame))
		return
	}

	transactionID = binary.BigEndian.Uint16(frame[0:2])
	protocolID := binary.BigEndian.Uint16(frame[2:4])
	length := binary.BigEndian.Uint16(frame[4:6])
	unitID = frame[6]
	pdu = frame[7:]

	if protocolID != ProtocolIdentifierTCP {
		err = fmt.Errorf("invalid protocol identifier: %d, expected %d", protocolID, ProtocolIdentifierTCP)
		return
	}

	expectedLength := uint16(len(pdu) + 1)
	if length != expectedLength {
		err = fmt.Errorf("invalid frame length in MBAP header: %d, expected %d", length, expectedLength)
		return
	}

	return
}