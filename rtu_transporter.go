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
	timeout     time.Duration
	packager    *RTUPackager
	port        io.ReadWriteCloser
	mu          sync.RWMutex // Protect concurrent access
	readBuffer  []byte       // Pre-allocated read buffer
	isTimedPort bool         // Whether port supports timeout operations
}

// TimedReadWriteCloser interface for ports that support timeout operations
type TimedReadWriteCloser interface {
	io.ReadWriteCloser
	SetReadTimeout(timeout time.Duration) error
	SetWriteTimeout(timeout time.Duration) error
}

// NewRTUTransporter creates a new RTUTransporter with optimizations
func NewRTUTransporter(port io.ReadWriteCloser, timeout time.Duration) *RTUTransporter {
	_, isTimedPort := port.(TimedReadWriteCloser)

	return &RTUTransporter{
		port:        port,
		timeout:     timeout,
		packager:    NewRTUPackager(),
		readBuffer:  make([]byte, 512), // Pre-allocate read buffer
		isTimedPort: isTimedPort,
	}
}

// SetTimeout updates the communication timeout
func (t *RTUTransporter) SetTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = timeout
}

// WriteRaw writes raw bytes to the serial port with timeout
func (t *RTUTransporter) WriteRaw(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot write empty data")
	}

	t.mu.RLock()
	timeout := t.timeout
	t.mu.RUnlock()

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
		_, err := t.port.Write(data)
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("write timeout after %v", timeout)
	}
}

// ReadRaw reads raw bytes from the serial port with timeout
func (t *RTUTransporter) ReadRaw() ([]byte, error) {
	t.mu.RLock()
	timeout := t.timeout
	t.mu.RUnlock()

	// Set read timeout if supported
	if t.isTimedPort {
		if timedPort, ok := t.port.(TimedReadWriteCloser); ok {
			if err := timedPort.SetReadTimeout(timeout); err != nil {
				return nil, fmt.Errorf("failed to set read timeout: %v", err)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type readResult struct {
		data []byte
		err  error
	}

	done := make(chan readResult, 1)
	go func() {
		n, err := t.port.Read(t.readBuffer)
		if err != nil {
			done <- readResult{nil, err}
			return
		}
		// Return a copy to avoid buffer reuse issues
		result := make([]byte, n)
		copy(result, t.readBuffer[:n])
		done <- readResult{result, nil}
	}()

	select {
	case result := <-done:
		return result.data, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("read timeout after %v", timeout)
	}
}

// Send sends a Modbus RTU PDU over the serial port
func (t *RTUTransporter) Send(slaveID uint8, pdu []byte) error {
	if len(pdu) == 0 {
		return fmt.Errorf("PDU cannot be empty")
	}

	frame, err := t.packager.Pack(slaveID, pdu)
	if err != nil {
		return fmt.Errorf("failed to pack frame: %v", err)
	}

	return t.WriteRaw(frame)
}

// readWithTimeout reads exactly n bytes with timeout
func (t *RTUTransporter) readWithTimeout(buffer []byte, n int) error {
	if n <= 0 || n > len(buffer) {
		return fmt.Errorf("invalid read size: %d", n)
	}

	t.mu.RLock()
	timeout := t.timeout
	t.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(t.port, buffer[:n])
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("read timeout after %v", timeout)
	}
}

// Receive receives a complete Modbus RTU frame with intelligent frame parsing
func (t *RTUTransporter) Receive() (uint8, []byte, error) {
	// Read header: Slave ID + Function Code
	header := make([]byte, 2)
	if err := t.readWithTimeout(header, 2); err != nil {
		return 0, nil, fmt.Errorf("failed to read header: %v", err)
	}

	functionCode := header[1]

	// Build frame starting with header
	frame := make([]byte, 2, 256)
	copy(frame, header)

	// Determine payload length based on function code
	payloadLength, err := t.getPayloadLength(functionCode)
	if err != nil {
		return 0, nil, err
	}

	// Read payload if needed
	if payloadLength > 0 {
		payload := make([]byte, payloadLength)
		if err := t.readWithTimeout(payload, payloadLength); err != nil {
			return 0, nil, fmt.Errorf("failed to read payload: %v", err)
		}
		frame = append(frame, payload...)
	}

	// Read CRC (2 bytes)
	crcBytes := make([]byte, 2)
	if err := t.readWithTimeout(crcBytes, 2); err != nil {
		return 0, nil, fmt.Errorf("failed to read CRC: %v", err)
	}
	frame = append(frame, crcBytes...)

	// Use RTUPackager to unpack and verify CRC
	unpackedSlaveID, pdu, err := t.packager.Unpack(frame)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to unpack frame: %v", err)
	}

	return unpackedSlaveID, pdu, nil
}

// getPayloadLength determines the expected payload length based on function code
func (t *RTUTransporter) getPayloadLength(functionCode uint8) (int, error) {
	switch functionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs,
		FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		// These responses have a byte count field, need to read it first
		countByte := make([]byte, 1)
		if err := t.readWithTimeout(countByte, 1); err != nil {
			return 0, fmt.Errorf("failed to read byte count: %v", err)
		}
		return int(countByte[0]), nil // +1 for the count byte itself

	case FuncCodeWriteSingleCoil, FuncCodeWriteSingleRegister:
		return 4, nil // Address (2) + Value (2)

	case FuncCodeWriteMultipleCoils, FuncCodeWriteMultipleRegisters:
		return 4, nil // Address (2) + Quantity (2)

	case FuncCodeReadExceptionStatus:
		return 1, nil // Status byte

	default:
		// For unknown function codes, try to handle as exception
		if functionCode >= 0x80 {
			return 1, nil // Exception code
		}
		return 0, fmt.Errorf("unsupported function code: 0x%02X", functionCode)
	}
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
