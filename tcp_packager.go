package modbus

import (
	"encoding/binary"
	"fmt"
)

// Modbus TCP Protocol Constants
const (
	TCPHeaderLength   = 7                              // MBAP header length in bytes
	MaxPDULength      = 253                            // Maximum PDU length according to Modbus spec
	MaxTCPFrameLength = TCPHeaderLength + MaxPDULength // Maximum complete frame length
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
	// Validate PDU length
	if len(pdu) == 0 {
		return nil, fmt.Errorf("PDU cannot be empty")
	}
	if len(pdu) > MaxPDULength {
		return nil, fmt.Errorf("PDU length %d exceeds maximum %d bytes", len(pdu), MaxPDULength)
	}

	// Length field includes the Unit Identifier (1 byte) + PDU length
	length := uint16(len(pdu) + 1)

	// Allocate frame buffer: MBAP header (7 bytes) + PDU
	frame := make([]byte, TCPHeaderLength+len(pdu))

	// Pack MBAP header
	binary.BigEndian.PutUint16(frame[0:2], transactionID)         // Transaction Identifier
	binary.BigEndian.PutUint16(frame[2:4], ProtocolIdentifierTCP) // Protocol Identifier
	binary.BigEndian.PutUint16(frame[4:6], length)                // Length
	frame[6] = unitID                                             // Unit Identifier

	// Copy PDU data
	copy(frame[7:], pdu)

	return frame, nil
}

// Unpack unpacks a Modbus TCP frame into a Transaction Identifier, Unit Identifier, and PDU.
func (p *TCPPackager) Unpack(frame []byte) (transactionID uint16, unitID uint8, pdu []byte, err error) {
	// Validate minimum frame length
	if len(frame) < TCPHeaderLength {
		err = fmt.Errorf("invalid TCP frame length: %d bytes, minimum required: %d bytes", len(frame), TCPHeaderLength)
		return
	}

	// Validate maximum frame length
	if len(frame) > MaxTCPFrameLength {
		err = fmt.Errorf("TCP frame length %d exceeds maximum %d bytes", len(frame), MaxTCPFrameLength)
		return
	}

	// Extract MBAP header fields
	transactionID = binary.BigEndian.Uint16(frame[0:2])
	protocolID := binary.BigEndian.Uint16(frame[2:4])
	length := binary.BigEndian.Uint16(frame[4:6])
	unitID = frame[6]

	// Validate protocol identifier
	if protocolID != ProtocolIdentifierTCP {
		err = fmt.Errorf("invalid protocol identifier: 0x%04X, expected 0x%04X", protocolID, ProtocolIdentifierTCP)
		return
	}

	// Validate length field
	if length == 0 {
		err = fmt.Errorf("invalid length field: cannot be zero")
		return
	}

	// Extract PDU
	pdu = frame[7:]

	// Validate that the length field matches actual frame structure
	// Length = Unit ID (1 byte) + PDU length
	expectedLength := uint16(len(pdu) + 1)
	if length != expectedLength {
		err = fmt.Errorf("length field mismatch: header indicates %d, actual frame has %d", length, expectedLength)
		return
	}

	// Validate PDU length doesn't exceed maximum
	if len(pdu) > MaxPDULength {
		err = fmt.Errorf("PDU length %d exceeds maximum %d bytes", len(pdu), MaxPDULength)
		return
	}

	return
}

// ValidateFrame performs basic validation on a TCP frame without full unpacking
func (p *TCPPackager) ValidateFrame(frame []byte) error {
	if len(frame) < TCPHeaderLength {
		return fmt.Errorf("frame too short: %d bytes, minimum: %d bytes", len(frame), TCPHeaderLength)
	}

	if len(frame) > MaxTCPFrameLength {
		return fmt.Errorf("frame too long: %d bytes, maximum: %d bytes", len(frame), MaxTCPFrameLength)
	}

	protocolID := binary.BigEndian.Uint16(frame[2:4])
	if protocolID != ProtocolIdentifierTCP {
		return fmt.Errorf("invalid protocol identifier: 0x%04X", protocolID)
	}

	length := binary.BigEndian.Uint16(frame[4:6])
	if length == 0 {
		return fmt.Errorf("invalid length field: cannot be zero")
	}

	expectedFrameLength := int(length) + 6 // Length field + Transaction ID + Protocol ID + Length field itself
	if len(frame) != expectedFrameLength {
		return fmt.Errorf("frame length mismatch: expected %d, got %d", expectedFrameLength, len(frame))
	}

	return nil
}
