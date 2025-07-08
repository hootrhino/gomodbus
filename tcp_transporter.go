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
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPTransporter handles Modbus TCP communication over a net.Conn.
type TCPTransporter struct {
	conn          net.Conn
	timeout       time.Duration
	packager      *TCPPackager
	logger        *log.Logger
	transactionID uint32       // Atomic counter for transaction IDs
	mu            sync.RWMutex // Protects connection operations
	closed        int32        // Atomic flag for closed state

	// Enhanced features
	maxRetries    int
	retryDelay    time.Duration
	keepAlive     bool
	keepAliveConf *KeepAliveConfig

	// Connection pooling support
	isPooled     bool
	poolReturnFn func()

	// Graceful shutdown support
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

// KeepAliveConfig holds TCP keep-alive configuration
type KeepAliveConfig struct {
	Enabled  bool
	Idle     time.Duration
	Interval time.Duration
	Count    int
}

// TCPTransporterConfig holds configuration for creating a TCPTransporter
type TCPTransporterConfig struct {
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	KeepAlive  *KeepAliveConfig
	Logger     io.Writer
}

// DefaultTCPTransporterConfig returns default configuration
func DefaultTCPTransporterConfig() TCPTransporterConfig {
	return TCPTransporterConfig{
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		KeepAlive: &KeepAliveConfig{
			Enabled:  true,
			Idle:     30 * time.Second,
			Interval: 30 * time.Second,
			Count:    3,
		},
	}
}

// NewTCPTransporter creates a new TCPTransporter with the given connection and configuration.
func NewTCPTransporter(conn net.Conn, config TCPTransporterConfig) *TCPTransporter {
	if config.Timeout == 0 {
		config.Timeout = DefaultTCPTransporterConfig().Timeout
	}

	var tcpLogger *log.Logger
	if config.Logger != nil {
		tcpLogger = log.New(config.Logger, "[TCP] ", log.LstdFlags|log.Lshortfile)
	}

	transporter := &TCPTransporter{
		conn:          conn,
		timeout:       config.Timeout,
		packager:      NewTCPPackager(),
		logger:        tcpLogger,
		transactionID: 0,
		maxRetries:    config.MaxRetries,
		retryDelay:    config.RetryDelay,
		shutdownCh:    make(chan struct{}),
	}

	// Configure keep-alive if enabled
	if config.KeepAlive != nil && config.KeepAlive.Enabled {
		transporter.keepAlive = true
		transporter.keepAliveConf = config.KeepAlive
		transporter.configureKeepAlive()
	}

	return transporter
}

// NewTCPTransporterSimple creates a new TCPTransporter with simple parameters (backward compatibility)
func NewTCPTransporterSimple(conn net.Conn, timeout time.Duration, logger io.Writer) *TCPTransporter {
	config := DefaultTCPTransporterConfig()
	config.Timeout = timeout
	config.Logger = logger
	return NewTCPTransporter(conn, config)
}

// configureKeepAlive sets up TCP keep-alive parameters
func (t *TCPTransporter) configureKeepAlive() {
	if tcpConn, ok := t.conn.(*net.TCPConn); ok && t.keepAliveConf != nil {
		if err := tcpConn.SetKeepAlive(t.keepAliveConf.Enabled); err != nil {
			t.log("Failed to set keep-alive: %v", err)
			return
		}

		if t.keepAliveConf.Enabled {
			if err := tcpConn.SetKeepAlivePeriod(t.keepAliveConf.Idle); err != nil {
				t.log("Failed to set keep-alive period: %v", err)
			}
		}
	}
}

// log writes a log message if logger is configured
func (t *TCPTransporter) log(format string, v ...any) {
	if t.logger != nil {
		t.logger.Printf(format, v...)
	}
}

// NextTransactionID generates the next transaction ID using atomic operations
func (t *TCPTransporter) NextTransactionID() uint16 {
	// Increment and wrap around at 65535 to avoid overflow
	id := atomic.AddUint32(&t.transactionID, 1)
	return uint16(id & 0xFFFF)
}

// setDeadline sets read/write deadline for the connection
func (t *TCPTransporter) setDeadline() error {
	if t.timeout > 0 {
		return t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	return nil
}

// clearDeadline clears the deadline on the connection
func (t *TCPTransporter) clearDeadline() {
	t.conn.SetDeadline(time.Time{})
}

// IsClosed returns whether the transporter is closed
func (t *TCPTransporter) IsClosed() bool {
	return atomic.LoadInt32(&t.closed) == 1
}

// WriteRaw writes raw bytes directly to the connection with enhanced error handling
func (t *TCPTransporter) WriteRaw(data []byte) error {
	return t.WriteRawWithContext(context.Background(), data)
}

// WriteRawWithContext writes raw bytes with context support
func (t *TCPTransporter) WriteRawWithContext(ctx context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.IsClosed() {
		return fmt.Errorf("transporter is closed")
	}

	if len(data) == 0 {
		return fmt.Errorf("no data to write")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t.log("Writing raw data: %d bytes", len(data))

	// Set deadline for write operation
	if err := t.setDeadline(); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	defer t.clearDeadline()

	// Write all data with context cancellation support
	written := 0
	for written < len(data) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := t.conn.Write(data[written:])
		if err != nil {
			return fmt.Errorf("write failed after %d bytes: %w", written, err)
		}
		written += n
	}

	t.log("Successfully wrote %d bytes", written)
	return nil
}

// ReadRaw reads raw bytes from the connection with enhanced buffering
func (t *TCPTransporter) ReadRaw() ([]byte, error) {
	return t.ReadRawWithContext(context.Background())
}

// ReadRawWithContext reads raw bytes with context support
func (t *TCPTransporter) ReadRawWithContext(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.IsClosed() {
		return nil, fmt.Errorf("transporter is closed")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use a buffer to handle maximum possible Modbus TCP frame
	buffer := make([]byte, MaxTCPFrameLength)

	t.log("Reading raw data from connection")

	// Set deadline for read operation
	if err := t.setDeadline(); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	defer t.clearDeadline()

	n, err := t.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	data := buffer[:n]
	t.log("Read %d bytes of raw data", n)

	return data, nil
}

// Send sends a Modbus TCP PDU over the connection with auto-generated transaction ID
func (t *TCPTransporter) Send(unitID uint8, pdu []byte) (uint16, error) {
	return t.SendWithContext(context.Background(), unitID, pdu)
}

// SendWithContext sends a Modbus TCP PDU with context support
func (t *TCPTransporter) SendWithContext(ctx context.Context, unitID uint8, pdu []byte) (uint16, error) {
	transactionID := t.NextTransactionID()
	err := t.SendWithTransactionIDAndContext(ctx, transactionID, unitID, pdu)
	return transactionID, err
}

// SendWithTransactionID sends a Modbus TCP PDU with a specific transaction ID
func (t *TCPTransporter) SendWithTransactionID(transactionID uint16, unitID uint8, pdu []byte) error {
	return t.SendWithTransactionIDAndContext(context.Background(), transactionID, unitID, pdu)
}

// SendWithTransactionIDAndContext sends a Modbus TCP PDU with context and specific transaction ID
func (t *TCPTransporter) SendWithTransactionIDAndContext(ctx context.Context, transactionID uint16, unitID uint8, pdu []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.IsClosed() {
		return fmt.Errorf("transporter is closed")
	}

	if len(pdu) == 0 {
		return fmt.Errorf("PDU cannot be empty")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t.log("Sending PDU: TxID=0x%04X, UnitID=%d, PDU length=%d", transactionID, unitID, len(pdu))

	// Pack the PDU into a TCP frame
	frame, err := t.packager.Pack(transactionID, unitID, pdu)
	if err != nil {
		return fmt.Errorf("failed to pack PDU: %w", err)
	}

	// Set deadline for write operation
	if err := t.setDeadline(); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	defer t.clearDeadline()

	// Write the complete frame with context cancellation support
	written := 0
	for written < len(frame) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := t.conn.Write(frame[written:])
		if err != nil {
			return fmt.Errorf("write failed after %d bytes: %w", written, err)
		}
		written += n
	}

	t.log("Successfully sent %d bytes (TxID=0x%04X)", written, transactionID)
	return nil
}

// Receive receives a complete Modbus TCP response from the connection
func (t *TCPTransporter) Receive() (transactionID uint16, unitID uint8, pdu []byte, err error) {
	return t.ReceiveWithContext(context.Background())
}

// ReceiveWithContext receives a complete Modbus TCP response with context support
func (t *TCPTransporter) ReceiveWithContext(ctx context.Context) (transactionID uint16, unitID uint8, pdu []byte, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.IsClosed() {
		err = fmt.Errorf("transporter is closed")
		return
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	default:
	}

	t.log("Receiving response from connection")

	// Set deadline for the entire receive operation
	if err = t.setDeadline(); err != nil {
		err = fmt.Errorf("failed to set read deadline: %w", err)
		return
	}
	defer t.clearDeadline()

	// Read MBAP Header (7 bytes) using io.ReadFull to ensure we get complete header
	header := make([]byte, TCPHeaderLength)
	if _, err = io.ReadFull(t.conn, header); err != nil {
		err = fmt.Errorf("failed to read MBAP header: %w", err)
		return
	}

	// Extract length from header to determine PDU size
	length := uint16(header[4])<<8 | uint16(header[5]) // Big-endian uint16

	// Validate length field
	if length == 0 {
		err = fmt.Errorf("invalid length field: cannot be zero")
		return
	}
	if length > MaxPDULength+1 {
		err = fmt.Errorf("length field too large: %d, maximum: %d", length, MaxPDULength+1)
		return
	}

	// Length includes Unit ID (1 byte), so PDU length is (length - 1)
	pduLength := int(length) - 1
	if pduLength < 0 {
		err = fmt.Errorf("invalid PDU length: %d", pduLength)
		return
	}

	// Read PDU data
	pduData := make([]byte, pduLength)
	if pduLength > 0 {
		if _, err = io.ReadFull(t.conn, pduData); err != nil {
			err = fmt.Errorf("failed to read PDU (%d bytes): %w", pduLength, err)
			return
		}
	}

	// Reconstruct complete frame for validation and unpacking
	completeFrame := make([]byte, TCPHeaderLength+pduLength)
	copy(completeFrame, header)
	copy(completeFrame[TCPHeaderLength:], pduData)

	// Unpack the complete frame
	transactionID, unitID, pdu, err = t.packager.Unpack(completeFrame)
	if err != nil {
		err = fmt.Errorf("failed to unpack frame: %w", err)
		return
	}

	t.log("Successfully received response: TxID=0x%04X, UnitID=%d, PDU length=%d",
		transactionID, unitID, len(pdu))

	return
}

// SendAndReceive sends a request and waits for a response with enhanced retry logic
func (t *TCPTransporter) SendAndReceive(unitID uint8, requestPDU []byte) (responsePDU []byte, err error) {
	return t.SendAndReceiveWithContext(context.Background(), unitID, requestPDU)
}

// SendAndReceiveWithContext sends a request and waits for a response with context support
func (t *TCPTransporter) SendAndReceiveWithContext(ctx context.Context, unitID uint8, requestPDU []byte) (responsePDU []byte, err error) {
	maxRetries := t.maxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Send the request
		txID, sendErr := t.SendWithContext(ctx, unitID, requestPDU)
		if sendErr != nil {
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("send failed after %d attempts: %w", maxRetries, sendErr)
			}
			t.log("Send attempt %d failed, retrying: %v", attempt+1, sendErr)

			// Wait before retry
			if t.retryDelay > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(t.retryDelay):
				}
			}
			continue
		}

		// Receive response with timeout
		respTxID, respUnitID, respPDU, receiveErr := t.ReceiveWithContext(ctx)
		if receiveErr != nil {
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("receive failed after %d attempts: %w", maxRetries, receiveErr)
			}
			t.log("Receive attempt %d failed, retrying: %v", attempt+1, receiveErr)

			// Wait before retry
			if t.retryDelay > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(t.retryDelay):
				}
			}
			continue
		}

		// Check if this is the response we're looking for
		if respTxID != txID {
			t.log("Transaction ID mismatch: sent=0x%04X, received=0x%04X, ignoring", txID, respTxID)
			continue
		}

		if respUnitID != unitID {
			t.log("Unit ID mismatch: sent=%d, received=%d, ignoring", unitID, respUnitID)
			continue
		}

		// Success
		return respPDU, nil
	}

	return nil, fmt.Errorf("no matching response received after %d attempts", maxRetries)
}

// Close closes the underlying connection and marks the transporter as closed
func (t *TCPTransporter) Close() error {
	// Use atomic operation to ensure thread-safe closing
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		return nil // Already closed
	}

	// Signal shutdown
	t.shutdownOnce.Do(func() {
		close(t.shutdownCh)
	})

	t.log("Closing TCP transporter")

	// Return connection to pool if it's pooled
	if t.isPooled && t.poolReturnFn != nil {
		t.poolReturnFn()
		return nil
	}

	if t.conn != nil {
		return t.conn.Close()
	}

	return nil
}

// GetLocalAddr returns the local network address
func (t *TCPTransporter) GetLocalAddr() net.Addr {
	if t.conn == nil {
		return nil
	}
	return t.conn.LocalAddr()
}

// GetRemoteAddr returns the remote network address
func (t *TCPTransporter) GetRemoteAddr() net.Addr {
	if t.conn == nil {
		return nil
	}
	return t.conn.RemoteAddr()
}

// SetPooled marks this transporter as being managed by a connection pool
func (t *TCPTransporter) SetPooled(returnFn func()) {
	t.isPooled = true
	t.poolReturnFn = returnFn
}

// GetShutdownChannel returns a channel that will be closed when the transporter is shutting down
func (t *TCPTransporter) GetShutdownChannel() <-chan struct{} {
	return t.shutdownCh
}

// Health check method to verify connection is still valid
func (t *TCPTransporter) HealthCheck() error {
	if t.IsClosed() {
		return fmt.Errorf("transporter is closed")
	}

	if t.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// Try to set a deadline as a simple connectivity test
	if err := t.conn.SetDeadline(time.Now().Add(time.Millisecond)); err != nil {
		return fmt.Errorf("connection health check failed: %w", err)
	}

	// Clear the deadline immediately
	t.conn.SetDeadline(time.Time{})
	return nil
}
