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

import "fmt"

// FreeFramePackager is a packager for arbitrary binary frames.
// It does not enforce any protocol structure or CRC.
type FreeFramePackager struct{}

// NewFreeFramePackager creates a new FreeFramePackager.
func NewFreeFramePackager() *FreeFramePackager {
	return &FreeFramePackager{}
}

// Pack simply returns a copy of the input data as the "frame".
func (p *FreeFramePackager) Pack(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}
	frame := make([]byte, len(data))
	copy(frame, data)
	return frame, nil
}

// Unpack simply returns a copy of the input frame as the "data".
func (p *FreeFramePackager) Unpack(frame []byte) ([]byte, error) {
	if len(frame) == 0 {
		return nil, fmt.Errorf("frame cannot be empty")
	}
	data := make([]byte, len(frame))
	copy(data, frame)
	return data, nil
}
