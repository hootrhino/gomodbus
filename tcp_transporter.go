package modbus

import (
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
	closed        bool
}

// NewTCPTransporter creates a new TCPTransporter with the given connection and timeout.
func NewTCPTransporter(conn net.Conn, timeout time.Duration, logger io.Writer) *TCPTransporter {
	var tcpLogger *log.Logger
	if logger != nil {
		tcpLogger = log.New(logger, "[TCP] ", log.LstdFlags|log.Lshortfile)
	}

	return &TCPTransporter{
		conn:          conn,
		timeout:       timeout,
		packager:      NewTCPPackager(),
		logger:        tcpLogger,
		transactionID: 0,
		closed:        false,
	}
}

// log writes a log message if logger is configured
func (t *TCPTransporter) log(format string, v ...interface{}) {
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

// WriteRaw writes raw bytes directly to the connection
func (t *TCPTransporter) WriteRaw(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transporter is closed")
	}

	if len(data) == 0 {
		return fmt.Errorf("no data to write")
	}

	t.log("Writing raw data: %d bytes", len(data))

	// Set deadline for write operation
	if err := t.setDeadline(); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	defer t.clearDeadline()

	// Write all data
	written := 0
	for written < len(data) {
		n, err := t.conn.Write(data[written:])
		if err != nil {
			return fmt.Errorf("write failed after %d bytes: %w", written, err)
		}
		written += n
	}

	t.log("Successfully wrote %d bytes", written)
	return nil
}

// ReadRaw reads raw bytes from the connection with proper buffering
func (t *TCPTransporter) ReadRaw() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, fmt.Errorf("transporter is closed")
	}

	// Use a larger buffer to handle maximum possible Modbus TCP frame
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
	transactionID := t.NextTransactionID()
	err := t.SendWithTransactionID(transactionID, unitID, pdu)
	return transactionID, err
}

// SendWithTransactionID sends a Modbus TCP PDU with a specific transaction ID
func (t *TCPTransporter) SendWithTransactionID(transactionID uint16, unitID uint8, pdu []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transporter is closed")
	}

	if len(pdu) == 0 {
		return fmt.Errorf("PDU cannot be empty")
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

	// Write the complete frame
	written := 0
	for written < len(frame) {
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
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		err = fmt.Errorf("transporter is closed")
		return
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

	// Validate the header using packager
	if err = t.packager.ValidateFrame(header); err != nil {
		// For header validation, we only check the first part
		// Create a temporary frame with minimum data for validation
		tempFrame := make([]byte, TCPHeaderLength+1)
		copy(tempFrame, header)
		// We'll validate the complete frame after reading PDU
	}

	// Extract length from header to determine PDU size
	length := uint16(header[4])<<8 | uint16(header[5]) // Big-endian uint16

	// Validate length field
	if length == 0 {
		err = fmt.Errorf("invalid length field: cannot be zero")
		return
	}
	if length > MaxPDULength+1 { // +1 for unit ID
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

// SendAndReceive sends a request and waits for a response with matching transaction ID
func (t *TCPTransporter) SendAndReceive(unitID uint8, requestPDU []byte) (responsePDU []byte, err error) {
	// Send the request
	txID, err := t.Send(unitID, requestPDU)
	if err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Receive response with timeout
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		respTxID, respUnitID, respPDU, receiveErr := t.Receive()

		if receiveErr != nil {
			if i == maxRetries-1 {
				return nil, fmt.Errorf("receive failed after %d retries: %w", maxRetries, receiveErr)
			}
			t.log("Receive attempt %d failed, retrying: %v", i+1, receiveErr)
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

		// This is our response
		return respPDU, nil
	}

	return nil, fmt.Errorf("no matching response received after %d attempts", maxRetries)
}

// Close closes the underlying connection and marks the transporter as closed
func (t *TCPTransporter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil // Already closed
	}

	t.closed = true
	t.log("Closing TCP transporter")

	if t.conn != nil {
		return t.conn.Close()
	}

	return nil
}

// IsClosed returns whether the transporter is closed
func (t *TCPTransporter) IsClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
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
