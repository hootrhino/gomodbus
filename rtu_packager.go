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
	"fmt"
)

// RTUPackager handles RTU frame packing/unpacking with CRC validation
type RTUPackager struct {
	crcTable [256]uint16 // Pre-calculated CRC table for faster computation
}

// NewRTUPackager creates a new RTU packager with pre-calculated CRC table
func NewRTUPackager() *RTUPackager {
	packager := &RTUPackager{}
	packager.initCRCTable()
	return packager
}

// initCRCTable initializes the CRC-16 lookup table (polynomial 0xA001)
func (p *RTUPackager) initCRCTable() {
	const polynomial = 0xA001 // CRC-16-ANSI polynomial (reversed)

	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc >>= 1
			}
		}
		p.crcTable[i] = crc
	}
}

// calculateCRC calculates CRC-16 for given data using lookup table
func (p *RTUPackager) calculateCRC(data []byte) uint16 {
	crc := uint16(0xFFFF) // Initial CRC value

	for _, b := range data {
		tableIndex := uint8(crc) ^ b
		crc = (crc >> 8) ^ p.crcTable[tableIndex]
	}

	return crc
}

// calculateCRCDirect calculates CRC-16 without lookup table (for verification)
func (p *RTUPackager) calculateCRCDirect(data []byte) uint16 {
	const polynomial = 0xA001
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}

// Pack creates an RTU frame with slave ID, PDU, and CRC
func (p *RTUPackager) Pack(slaveID uint8, pdu []byte) ([]byte, error) {
	if slaveID == 0 {
		return nil, fmt.Errorf("slaveID cannot be zero")
	}
	if len(pdu) == 0 {
		return nil, fmt.Errorf("PDU cannot be empty")
	}

	if len(pdu) > 253 {
		return nil, fmt.Errorf("PDU too long: %d bytes (max 253)", len(pdu))
	}

	// Validate slave ID
	if slaveID > 247 {
		return nil, fmt.Errorf("invalid slave ID: %d (must be 1-247)", slaveID)
	}

	// Create frame: SlaveID + PDU + CRC
	frameLen := 1 + len(pdu) + 2
	frame := make([]byte, frameLen)

	// Add slave ID
	frame[0] = slaveID

	// Add PDU
	copy(frame[1:], pdu)

	// Calculate and add CRC
	crc := p.calculateCRC(frame[:frameLen-2])

	// CRC is transmitted in little-endian format
	frame[frameLen-2] = byte(crc & 0xFF)        // Low byte first
	frame[frameLen-1] = byte((crc >> 8) & 0xFF) // High byte second

	return frame, nil
}

// Unpack extracts slave ID and PDU from RTU frame with CRC validation
func (p *RTUPackager) Unpack(frame []byte) (uint8, []byte, error) {
	if len(frame) < 4 {
		return 0, nil, fmt.Errorf("frame too short: %d bytes (minimum 4)", len(frame))
	}

	// Verify CRC
	if !p.VerifyCRC(frame) {
		return 0, nil, fmt.Errorf("CRC verification failed")
	}

	// Extract slave ID
	slaveID := frame[0]

	// Extract PDU (everything except slave ID and CRC)
	pdu := make([]byte, len(frame)-3)
	copy(pdu, frame[1:len(frame)-2])

	return slaveID, pdu, nil
}

// VerifyCRC verifies the CRC of an RTU frame
func (p *RTUPackager) VerifyCRC(frame []byte) bool {
	if len(frame) < 4 {
		return false
	}

	// Calculate CRC for data (excluding the CRC bytes)
	dataLen := len(frame) - 2
	calculatedCRC := p.calculateCRC(frame[:dataLen])

	// Extract CRC from frame (little-endian)
	receivedCRC := uint16(frame[dataLen]) | (uint16(frame[dataLen+1]) << 8)

	return calculatedCRC == receivedCRC
}

// VerifyCRCWithDetails verifies CRC and returns detailed information
func (p *RTUPackager) VerifyCRCWithDetails(frame []byte) (bool, uint16, uint16, error) {
	if len(frame) < 4 {
		return false, 0, 0, fmt.Errorf("frame too short: %d bytes", len(frame))
	}

	// Calculate CRC for data
	dataLen := len(frame) - 2
	calculatedCRC := p.calculateCRC(frame[:dataLen])

	// Extract received CRC
	receivedCRC := uint16(frame[dataLen]) | (uint16(frame[dataLen+1]) << 8)

	return calculatedCRC == receivedCRC, calculatedCRC, receivedCRC, nil
}

// ValidateFrame performs comprehensive frame validation
func (p *RTUPackager) ValidateFrame(frame []byte) error {
	if len(frame) < 4 {
		return fmt.Errorf("frame too short: %d bytes (minimum 4)", len(frame))
	}

	if len(frame) > 256 {
		return fmt.Errorf("frame too long: %d bytes (maximum 256)", len(frame))
	}

	// Validate slave ID
	slaveID := frame[0]
	if slaveID > 247 {
		return fmt.Errorf("invalid slave ID: %d (must be 1-247)", slaveID)
	}

	// Validate function code
	functionCode := frame[1]
	if functionCode == 0 {
		return fmt.Errorf("invalid function code: 0")
	}

	// Validate CRC
	isValid, calculated, received, err := p.VerifyCRCWithDetails(frame)
	if err != nil {
		return fmt.Errorf("CRC validation error: %v", err)
	}

	if !isValid {
		return fmt.Errorf("CRC mismatch: calculated=0x%04X, received=0x%04X", calculated, received)
	}

	return nil
}

// RepairFrame attempts to repair a frame with CRC errors (if possible)
func (p *RTUPackager) RepairFrame(frame []byte) ([]byte, error) {
	if len(frame) < 4 {
		return nil, fmt.Errorf("frame too short for repair")
	}

	// Create a copy of the frame
	repairedFrame := make([]byte, len(frame))
	copy(repairedFrame, frame)

	// Recalculate and replace CRC
	dataLen := len(repairedFrame) - 2
	crc := p.calculateCRC(repairedFrame[:dataLen])

	repairedFrame[dataLen] = byte(crc & 0xFF)
	repairedFrame[dataLen+1] = byte((crc >> 8) & 0xFF)

	return repairedFrame, nil
}

// GetCRCBytes returns the CRC bytes for given data
func (p *RTUPackager) GetCRCBytes(data []byte) []byte {
	crc := p.calculateCRC(data)
	return []byte{byte(crc & 0xFF), byte((crc >> 8) & 0xFF)}
}

// CompareCRCMethods compares lookup table vs direct calculation (for testing)
func (p *RTUPackager) CompareCRCMethods(data []byte) (uint16, uint16, bool) {
	tableCRC := p.calculateCRC(data)
	directCRC := p.calculateCRCDirect(data)
	return tableCRC, directCRC, tableCRC == directCRC
}

// GetFrameInfo returns detailed information about a frame
func (p *RTUPackager) GetFrameInfo(frame []byte) map[string]interface{} {
	info := make(map[string]interface{})

	if len(frame) < 4 {
		info["error"] = "frame too short"
		return info
	}

	info["length"] = len(frame)
	info["slave_id"] = frame[0]
	info["function_code"] = frame[1]

	// CRC information
	isValid, calculated, received, err := p.VerifyCRCWithDetails(frame)
	if err != nil {
		info["crc_error"] = err.Error()
	} else {
		info["crc_valid"] = isValid
		info["crc_calculated"] = fmt.Sprintf("0x%04X", calculated)
		info["crc_received"] = fmt.Sprintf("0x%04X", received)
	}

	// PDU information
	if len(frame) > 3 {
		pduLen := len(frame) - 3
		info["pdu_length"] = pduLen
		info["pdu_hex"] = fmt.Sprintf("% X", frame[1:len(frame)-2])
	}

	// Frame validation
	if validationErr := p.ValidateFrame(frame); validationErr != nil {
		info["validation_error"] = validationErr.Error()
	} else {
		info["validation_status"] = "valid"
	}

	return info
}

// DumpFrame returns a hex dump of the frame with annotations
func (p *RTUPackager) DumpFrame(frame []byte) string {
	if len(frame) == 0 {
		return "Empty frame"
	}

	var result string
	result += fmt.Sprintf("Frame Length: %d bytes\n", len(frame))
	result += fmt.Sprintf("Hex: % X\n", frame)

	if len(frame) >= 1 {
		result += fmt.Sprintf("Slave ID: %d (0x%02X)\n", frame[0], frame[0])
	}

	if len(frame) >= 2 {
		functionCode := frame[1]
		result += fmt.Sprintf("Function Code: %d (0x%02X)", functionCode, functionCode)
		if functionCode >= 0x80 {
			result += " [Exception Response]"
		}
		result += "\n"
	}

	if len(frame) >= 4 {
		pduLen := len(frame) - 3
		result += fmt.Sprintf("PDU Length: %d bytes\n", pduLen)
		result += fmt.Sprintf("PDU: % X\n", frame[1:len(frame)-2])

		// CRC information
		dataLen := len(frame) - 2
		calculated := p.calculateCRC(frame[:dataLen])
		received := uint16(frame[dataLen]) | (uint16(frame[dataLen+1]) << 8)

		result += fmt.Sprintf("CRC Calculated: 0x%04X\n", calculated)
		result += fmt.Sprintf("CRC Received: 0x%04X\n", received)
		result += fmt.Sprintf("CRC Valid: %t\n", calculated == received)
	}

	return result
}
