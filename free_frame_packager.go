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
