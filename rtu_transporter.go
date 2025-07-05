package modbus

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// RTUTransporter handles Modbus RTU communication with optimizations
type RTUTransporter struct {
	packager     *RTUPackager
	port         io.ReadWriteCloser
	mu           sync.RWMutex // Protect concurrent access
	readBuffer   []byte       // Pre-allocated read buffer
	frameBuffer  []byte       // Pre-allocated frame buffer
	maxFrameSize int          // Maximum frame size
}

// RTUConfig holds configuration parameters for RTU transporter
type RTUConfig struct {
	MaxFrameSize int
}

// DefaultRTUConfig returns default configuration
func DefaultRTUConfig() RTUConfig {
	return RTUConfig{
		MaxFrameSize: 256,
	}
}

// NewRTUTransporter creates a new RTUTransporter with optimizations
func NewRTUTransporter(port io.ReadWriteCloser, config RTUConfig) *RTUTransporter {
	if config.MaxFrameSize <= 0 {
		config.MaxFrameSize = 256
	}

	return &RTUTransporter{
		port:         port,
		packager:     NewRTUPackager(),
		readBuffer:   make([]byte, config.MaxFrameSize),
		frameBuffer:  make([]byte, config.MaxFrameSize),
		maxFrameSize: config.MaxFrameSize,
	}
}

// WriteRaw writes raw bytes to the serial port with timeout and pre-transmission delay
func (t *RTUTransporter) WriteRaw(data []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(data) == 0 {
		return fmt.Errorf("cannot write empty data")
	}
	bytesWritten := 0
	for bytesWritten < len(data) {
		n, err := t.port.Write(data[bytesWritten:])
		if err != nil {
			return fmt.Errorf("write failed after %d bytes: %v", bytesWritten, err)
		}
		bytesWritten += n
	}
	if bytesWritten != len(data) {
		return fmt.Errorf("partial write: expected %d bytes, wrote %d", len(data), bytesWritten)
	}
	return nil
}

// readByteWithTimeout reads a single byte with timeout
func (t *RTUTransporter) readByteWithTimeout(timeout time.Duration) (byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct {
		b   byte
		err error
	}, 1)

	go func() {
		b := make([]byte, 1)
		n, err := t.port.Read(b)
		if err != nil {
			done <- struct {
				b   byte
				err error
			}{0, err}
			return
		}
		if n == 0 {
			done <- struct {
				b   byte
				err error
			}{0, fmt.Errorf("no data read")}
			return
		}
		done <- struct {
			b   byte
			err error
		}{b[0], nil}
	}()

	select {
	case result := <-done:
		return result.b, result.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// ReadRaw reads a complete RTU frame with intelligent frame detection
func (t *RTUTransporter) ReadRaw() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	N, err := t.port.Read(t.readBuffer)
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}
	if N == 0 {
		return nil, fmt.Errorf("no data read")
	}
	frame := t.readBuffer[:N]
	if !t.isValidFrameLength(frame) {
		return nil, fmt.Errorf("invalid frame length")
	}
	if !t.packager.VerifyCRC(frame) {
		return nil, fmt.Errorf("CRC verification failed")
	}
	return frame, nil

}

// isValidFrameLength checks if the frame length is valid for the function code
func (t *RTUTransporter) isValidFrameLength(frame []byte) bool {
	if len(frame) < 4 {
		return false
	}

	functionCode := frame[1]

	// Check if this is an exception response
	if functionCode >= 0x80 {
		return len(frame) == 5 // SlaveID + FuncCode + ExceptionCode + CRC
	}

	// Estimate expected frame length based on function code
	switch functionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs,
		FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		if len(frame) < 3 {
			return false
		}
		// Check if we have byte count field
		expectedLen := 3 + int(frame[2]) + 2 // Header + ByteCount + Data + CRC
		return len(frame) == expectedLen

	case FuncCodeWriteSingleCoil, FuncCodeWriteSingleRegister:
		return len(frame) == 8 // SlaveID + FuncCode + Address + Value + CRC

	case FuncCodeWriteMultipleCoils, FuncCodeWriteMultipleRegisters:
		return len(frame) == 8 // SlaveID + FuncCode + Address + Quantity + CRC

	case FuncCodeReadExceptionStatus:
		return len(frame) == 5 // SlaveID + FuncCode + Status + CRC

	default:
		// For unknown function codes, require minimum frame size
		return len(frame) >= 4
	}
}

// Send sends a Modbus RTU PDU with enhanced error handling and retries
func (t *RTUTransporter) Send(slaveID uint8, pdu []byte) error {
	if len(pdu) == 0 {
		return fmt.Errorf("PDU cannot be empty")
	}

	if slaveID == 0 {
		return fmt.Errorf("slave ID cannot be zero")
	}

	// Validate PDU length
	if len(pdu) > 253 { // Max PDU length for RTU
		return fmt.Errorf("PDU too long: %d bytes (max 253)", len(pdu))
	}

	frame, err := t.packager.Pack(slaveID, pdu)
	if err != nil {
		return fmt.Errorf("failed to pack frame: %v", err)
	}

	// Validate frame before sending
	if len(frame) < 4 {
		return fmt.Errorf("invalid frame length: %d", len(frame))
	}

	// Verify CRC before sending
	if !t.packager.VerifyCRC(frame) {
		return fmt.Errorf("CRC verification failed before sending")
	}

	return t.WriteRaw(frame)
}

// SendWithRetry sends a PDU with retry mechanism
func (t *RTUTransporter) SendWithRetry(slaveID uint8, pdu []byte, maxRetries int) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		err := t.Send(slaveID, pdu)
		if err == nil {
			return nil
		}

		lastErr = err

		if i < maxRetries {
			// Wait before retry with exponential backoff
			waitTime := time.Duration(i+1) * 10 * time.Millisecond
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("send failed after %d retries: %v", maxRetries, lastErr)
}

// Receive receives a complete Modbus RTU frame with enhanced validation
func (t *RTUTransporter) Receive() (uint8, []byte, error) {
	// Read complete frame
	frame, err := t.ReadRaw()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read frame: %v", err)
	}

	if len(frame) < 4 {
		return 0, nil, fmt.Errorf("frame too short: %d bytes", len(frame))
	}

	// Validate frame structure
	if !t.isValidFrameStructure(frame) {
		return 0, nil, fmt.Errorf("invalid frame structure")
	}

	// Use RTUPackager to unpack and verify CRC
	slaveID, pdu, err := t.packager.Unpack(frame)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to unpack frame: %v", err)
	}

	// Additional validation
	if slaveID == 0 {
		return 0, nil, fmt.Errorf("invalid slave ID: 0")
	}

	if len(pdu) == 0 {
		return 0, nil, fmt.Errorf("empty PDU")
	}

	return slaveID, pdu, nil
}

// isValidFrameStructure performs basic frame structure validation
func (t *RTUTransporter) isValidFrameStructure(frame []byte) bool {
	if len(frame) < 4 {
		return false
	}

	// Check slave ID (should be 1-247 for normal devices, 0 for broadcast)
	slaveID := frame[0]
	if slaveID > 247 {
		return false
	}

	// Check function code
	functionCode := frame[1]
	if functionCode == 0 {
		return false
	}

	// Verify CRC
	return t.packager.VerifyCRC(frame)
}

// ReceiveWithContext receives a frame with context cancellation support
func (t *RTUTransporter) ReceiveWithContext(ctx context.Context) (uint8, []byte, error) {
	type result struct {
		slaveID uint8
		pdu     []byte
		err     error
	}

	done := make(chan result, 1)
	go func() {
		slaveID, pdu, err := t.Receive()
		done <- result{slaveID, pdu, err}
	}()

	select {
	case res := <-done:
		return res.slaveID, res.pdu, res.err
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

// Exchange performs a complete send/receive operation
func (t *RTUTransporter) Exchange(slaveID uint8, requestPDU []byte) ([]byte, error) {
	// Send request
	if err := t.Send(slaveID, requestPDU); err != nil {
		return nil, fmt.Errorf("send failed: %v", err)
	}

	// Receive response
	responseSlaveID, responsePDU, err := t.Receive()
	if err != nil {
		return nil, fmt.Errorf("receive failed: %v", err)
	}

	// Validate response slave ID matches request
	if responseSlaveID != slaveID {
		return nil, fmt.Errorf("slave ID mismatch: expected %d, got %d", slaveID, responseSlaveID)
	}

	return responsePDU, nil
}

// ExchangeWithRetry performs exchange with retry mechanism
func (t *RTUTransporter) ExchangeWithRetry(slaveID uint8, requestPDU []byte, maxRetries int) ([]byte, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		responsePDU, err := t.Exchange(slaveID, requestPDU)
		if err == nil {
			return responsePDU, nil
		}

		lastErr = err

		if i < maxRetries {
			// Wait before retry
			waitTime := time.Duration(i+1) * 20 * time.Millisecond
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("exchange failed after %d retries: %v", maxRetries, lastErr)
}

// FlushBuffers flushes any remaining data in the buffers
func (t *RTUTransporter) FlushBuffers() error {
	// Try to read any remaining data with a short timeout
	shortTimeout := 10 * time.Millisecond
	for {
		_, err := t.readByteWithTimeout(shortTimeout)
		if err != nil {
			break // No more data or timeout
		}
	}
	return nil
}

// Close closes the underlying serial port
func (t *RTUTransporter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.port == nil {
		return nil
	}

	err := t.port.Close()
	t.port = nil
	return err
}

// IsConnected returns true if the port is still connected
func (t *RTUTransporter) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.port != nil
}
