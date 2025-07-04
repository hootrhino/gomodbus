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
	timeout       time.Duration
	packager      *RTUPackager
	port          io.ReadWriteCloser
	mu            sync.RWMutex  // Protect concurrent access
	readBuffer    []byte        // Pre-allocated read buffer
	frameBuffer   []byte        // Pre-allocated frame buffer
	isTimedPort   bool          // Whether port supports timeout operations
	maxFrameSize  int           // Maximum frame size
	interCharTime time.Duration // Inter-character timeout
	frameTimeout  time.Duration // Frame timeout
}

// TimedReadWriteCloser interface for ports that support timeout operations
type TimedReadWriteCloser interface {
	io.ReadWriteCloser
	SetReadTimeout(timeout time.Duration) error
	SetWriteTimeout(timeout time.Duration) error
}

// RTUConfig holds configuration parameters for RTU transporter
type RTUConfig struct {
	Timeout       time.Duration
	InterCharTime time.Duration
	FrameTimeout  time.Duration
	MaxFrameSize  int
}

// DefaultRTUConfig returns default configuration
func DefaultRTUConfig() RTUConfig {
	return RTUConfig{
		Timeout:       1 * time.Second,
		InterCharTime: 3 * time.Millisecond, // 3.5 chars at 9600 baud
		FrameTimeout:  100 * time.Millisecond,
		MaxFrameSize:  256,
	}
}

// NewRTUTransporter creates a new RTUTransporter with optimizations
func NewRTUTransporter(port io.ReadWriteCloser, config RTUConfig) *RTUTransporter {
	_, isTimedPort := port.(TimedReadWriteCloser)

	if config.MaxFrameSize <= 0 {
		config.MaxFrameSize = 256
	}

	return &RTUTransporter{
		port:          port,
		timeout:       config.Timeout,
		interCharTime: config.InterCharTime,
		frameTimeout:  config.FrameTimeout,
		packager:      NewRTUPackager(),
		readBuffer:    make([]byte, config.MaxFrameSize),
		frameBuffer:   make([]byte, config.MaxFrameSize),
		isTimedPort:   isTimedPort,
		maxFrameSize:  config.MaxFrameSize,
	}
}

// SetTimeout updates the communication timeout
func (t *RTUTransporter) SetTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = timeout
}

// WriteRaw writes raw bytes to the serial port with timeout and pre-transmission delay
func (t *RTUTransporter) WriteRaw(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot write empty data")
	}

	t.mu.RLock()
	timeout := t.timeout
	t.mu.RUnlock()

	// Ensure minimum inter-frame delay before transmission
	time.Sleep(t.interCharTime)

	// Set write timeout if supported
	if t.isTimedPort {
		if timedPort, ok := t.port.(TimedReadWriteCloser); ok {
			if err := timedPort.SetWriteTimeout(timeout); err != nil {
				return fmt.Errorf("failed to set write timeout: %v", err)
			}
		}
	}

	// Use context for timeout control
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Write all data at once to minimize transmission delays
		bytesWritten := 0
		for bytesWritten < len(data) {
			n, err := t.port.Write(data[bytesWritten:])
			if err != nil {
				done <- fmt.Errorf("write failed after %d bytes: %v", bytesWritten, err)
				return
			}
			bytesWritten += n
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("write timeout after %v", timeout)
	}
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
	timeout := t.timeout
	interCharTime := t.interCharTime
	t.mu.RUnlock()

	// Set read timeout if supported
	if t.isTimedPort {
		if timedPort, ok := t.port.(TimedReadWriteCloser); ok {
			if err := timedPort.SetReadTimeout(timeout); err != nil {
				return nil, fmt.Errorf("failed to set read timeout: %v", err)
			}
		}
	}

	var frame []byte
	frameStarted := false
	lastByteTime := time.Now()

	// Overall timeout for the entire frame
	frameCtx, frameCancel := context.WithTimeout(context.Background(), timeout)
	defer frameCancel()

	for {
		// Check if overall timeout exceeded
		select {
		case <-frameCtx.Done():
			if len(frame) > 0 {
				return frame, nil // Return partial frame
			}
			return nil, fmt.Errorf("frame timeout after %v", timeout)
		default:
		}

		// Read next byte with inter-character timeout
		readTimeout := interCharTime
		if !frameStarted {
			readTimeout = timeout // First byte can take longer
		}

		b, err := t.readByteWithTimeout(readTimeout)
		if err != nil {
			// If we have a partial frame and timeout, consider frame complete
			if len(frame) > 0 && (err == context.DeadlineExceeded || err.Error() == "context deadline exceeded") {
				return frame, nil
			}
			return nil, fmt.Errorf("read error: %v", err)
		}

		currentTime := time.Now()

		// Check for inter-character timeout (frame boundary detection)
		if frameStarted && currentTime.Sub(lastByteTime) > interCharTime {
			// Inter-character timeout exceeded, previous frame is complete
			// Start new frame with current byte
			frame = []byte{b}
		} else {
			// Add byte to current frame
			frame = append(frame, b)
		}

		frameStarted = true
		lastByteTime = currentTime

		// Check if we have a minimum valid frame (at least 4 bytes: SlaveID + FuncCode + CRC)
		if len(frame) >= 4 {
			// Try to validate if this could be a complete frame
			if t.isValidFrameLength(frame) {
				// Wait a bit more to see if more data arrives
				time.Sleep(interCharTime / 2)

				// Try to read one more byte to confirm frame end
				_, err := t.readByteWithTimeout(interCharTime / 2)
				if err != nil {
					// No more data, frame is complete
					return frame, nil
				}
			}
		}

		// Prevent infinite frames
		if len(frame) >= t.maxFrameSize {
			return frame, nil
		}
	}
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

// ReceiveWithTimeout receives a frame with specific timeout
func (t *RTUTransporter) ReceiveWithTimeout(timeout time.Duration) (uint8, []byte, error) {
	// Temporarily set timeout
	originalTimeout := t.timeout
	t.SetTimeout(timeout)
	defer t.SetTimeout(originalTimeout)

	return t.Receive()
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

// GetStats returns communication statistics
func (t *RTUTransporter) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"timeout":         t.timeout,
		"inter_char_time": t.interCharTime,
		"frame_timeout":   t.frameTimeout,
		"max_frame_size":  t.maxFrameSize,
		"is_timed_port":   t.isTimedPort,
	}
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
