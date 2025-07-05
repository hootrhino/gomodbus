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
	"bytes"
	"testing"
)

func TestTCPPackager_PackUnpack(t *testing.T) {
	p := NewTCPPackager()
	transactionID := uint16(0x1234)
	unitID := uint8(0x01)
	pdu := []byte{0x03, 0x00, 0x00, 0x00, 0x01}

	frame, err := p.Pack(transactionID, unitID, pdu)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	gotTID, gotUID, gotPDU, err := p.Unpack(frame)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if gotTID != transactionID {
		t.Errorf("transactionID mismatch: got %04x, want %04x", gotTID, transactionID)
	}
	if gotUID != unitID {
		t.Errorf("unitID mismatch: got %02x, want %02x", gotUID, unitID)
	}
	if !bytes.Equal(gotPDU, pdu) {
		t.Errorf("PDU mismatch: got %v, want %v", gotPDU, pdu)
	}
}

func TestTCPPackager_Pack_Invalid(t *testing.T) {
	p := NewTCPPackager()
	_, err := p.Pack(1, 1, nil)
	if err == nil {
		t.Error("Pack should fail for empty PDU")
	}
	_, err = p.Pack(1, 1, make([]byte, MaxPDULength+1))
	if err == nil {
		t.Error("Pack should fail for PDU exceeding max length")
	}
}

func TestTCPPackager_Unpack_Invalid(t *testing.T) {
	p := NewTCPPackager()
	// Too short
	_, _, _, err := p.Unpack([]byte{1, 2, 3})
	if err == nil {
		t.Error("Unpack should fail for short frame")
	}
	// Too long
	_, _, _, err = p.Unpack(make([]byte, MaxTCPFrameLength+1))
	if err == nil {
		t.Error("Unpack should fail for long frame")
	}
	// Invalid protocol ID
	frame, _ := p.Pack(1, 1, []byte{0x03, 0x00})
	frame[2] = 0xFF
	frame[3] = 0xFF
	_, _, _, err = p.Unpack(frame)
	if err == nil {
		t.Error("Unpack should fail for invalid protocol ID")
	}
	// Invalid length field
	frame, _ = p.Pack(1, 1, []byte{0x03, 0x00})
	frame[4] = 0x00
	frame[5] = 0x00
	_, _, _, err = p.Unpack(frame)
	if err == nil {
		t.Error("Unpack should fail for zero length field")
	}
}

func TestTCPPackager_ValidateFrame(t *testing.T) {
	p := NewTCPPackager()
	frame, _ := p.Pack(1, 1, []byte{0x03, 0x00})
	if err := p.ValidateFrame(frame); err != nil {
		t.Errorf("ValidateFrame failed for valid frame: %v", err)
	}
	// Short frame
	if err := p.ValidateFrame([]byte{1, 2, 3}); err == nil {
		t.Error("ValidateFrame should fail for short frame")
	}
	// Long frame
	if err := p.ValidateFrame(make([]byte, MaxTCPFrameLength+1)); err == nil {
		t.Error("ValidateFrame should fail for long frame")
	}
	// Invalid protocol ID
	frame[2] = 0xFF
	frame[3] = 0xFF
	if err := p.ValidateFrame(frame); err == nil {
		t.Error("ValidateFrame should fail for invalid protocol ID")
	}
}
