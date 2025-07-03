package modbus

import (
	"fmt"
)

// RTUPackager handles Modbus RTU packet packing and unpacking with optimizations
type RTUPackager struct {
	// Pre-allocated buffer for frame construction to reduce allocations
	frameBuffer []byte
}

// NewRTUPackager creates a new RTUPackager with pre-allocated buffer
func NewRTUPackager() *RTUPackager {
	return &RTUPackager{
		frameBuffer: make([]byte, 256), // Pre-allocate reasonable size buffer
	}
}

// Pack packs a Modbus RTU PDU into a complete RTU frame
// The RTU frame format is: Slave Address (1 byte) + Function Code (1 byte) + Data (variable length) + CRC (2 bytes, big-endian)
func (p *RTUPackager) Pack(slaveID uint8, pdu []byte) ([]byte, error) {
	if len(pdu) == 0 {
		return nil, fmt.Errorf("PDU cannot be empty")
	}

	frameLength := 1 + len(pdu) + 2 // slave ID + PDU + CRC

	// Resize buffer if needed
	if frameLength > len(p.frameBuffer) {
		p.frameBuffer = make([]byte, frameLength*2) // Allocate with some headroom
	}

	frame := p.frameBuffer[:frameLength]
	frame[0] = slaveID
	copy(frame[1:], pdu)

	// Calculate CRC for the frame without CRC bytes
	crc := CRC16(frame[:frameLength-2])

	// Write CRC as big-endian (high byte first, then low byte)
	frame[frameLength-2] = byte((crc >> 8) & 0xFF) // CRC high byte
	frame[frameLength-1] = byte(crc & 0xFF)        // CRC low byte

	// Return a copy to avoid buffer reuse issues
	result := make([]byte, frameLength)
	copy(result, frame)
	return result, nil
}

// Unpack unpacks a Modbus RTU frame into Slave Address and PDU with CRC verification (big-endian CRC)
func (p *RTUPackager) Unpack(frame []byte) (slaveID uint8, pdu []byte, err error) {
	// Minimum RTU frame: Slave ID (1) + Function Code (1) + CRC (2) = 4 bytes
	if len(frame) < 4 {
		return 0, nil, fmt.Errorf("invalid RTU frame length: %d bytes, minimum is 4", len(frame))
	}

	// Extract CRC from frame (big-endian)
	frameLen := len(frame)
	receivedCRC := (uint16(frame[frameLen-2]) << 8) | uint16(frame[frameLen-1])

	// Calculate CRC for frame without CRC bytes
	calculatedCRC := CRC16(frame[:frameLen-2])

	if receivedCRC != calculatedCRC {
		return 0, nil, fmt.Errorf("CRC mismatch: received 0x%04X, calculated 0x%04X", receivedCRC, calculatedCRC)
	}

	slaveID = frame[0]
	pdu = frame[1 : frameLen-2]
	return slaveID, pdu, nil
}
