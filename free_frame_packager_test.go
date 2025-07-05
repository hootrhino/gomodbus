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

func TestFreeFramePackager_PackUnpack(t *testing.T) {
	p := NewFreeFramePackager()
	data := []byte{0x01, 0x02, 0x03, 0xFF}

	// Test Pack
	frame, err := p.Pack(data)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if !bytes.Equal(frame, data) {
		t.Errorf("Pack returned %v, want %v", frame, data)
	}
	// Ensure it's a copy, not the same slice
	frame[0] = 0x99
	if data[0] == 0x99 {
		t.Error("Pack did not return a copy of the input data")
	}

	// Test Unpack
	unpacked, err := p.Unpack(frame)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if !bytes.Equal(unpacked, frame) {
		t.Errorf("Unpack returned %v, want %v", unpacked, frame)
	}
	// Ensure it's a copy, not the same slice
	unpacked[0] = 0x55
	if frame[0] == 0x55 {
		t.Error("Unpack did not return a copy of the input frame")
	}
}

func TestFreeFramePackager_Pack_Empty(t *testing.T) {
	p := NewFreeFramePackager()
	_, err := p.Pack([]byte{})
	if err == nil {
		t.Error("Pack should fail for empty data")
	}
}

func TestFreeFramePackager_Unpack_Empty(t *testing.T) {
	p := NewFreeFramePackager()
	_, err := p.Unpack([]byte{})
	if err == nil {
		t.Error("Unpack should fail for empty frame")
	}
}
