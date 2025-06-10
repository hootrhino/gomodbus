package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// TCPTransporter handles Modbus TCP communication over a net.Conn.
type TCPTransporter struct {
	conn     net.Conn
	timeout  time.Duration
	packager *TCPPackager
}

// NewTCPTransporter creates a new TCPTransporter with the given connection and timeout.
func NewTCPTransporter(conn net.Conn, timeout time.Duration, logger io.Writer) *TCPTransporter {
	return &TCPTransporter{
		conn:     conn,
		timeout:  timeout,
		packager: NewTCPPackager(),
	}
}

func (t *TCPTransporter) WriteRaw(pdu []byte) error {
	// Set timeout for write operation
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return err
	}
	defer t.conn.SetDeadline(time.Time{}) // Reset deadline after write
	// Write the raw PDU to the connection
	_, err := t.conn.Write(pdu)
	return err
}

// ReadRaw reads a raw bytes serial port.
func (t *TCPTransporter) ReadRaw() ([]byte, error) {
	buffer := make([]byte, 64) // Adjust size as needed
	// Set timeout for read operation
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	defer t.conn.SetDeadline(time.Time{}) // Reset deadline after read
	n, err := t.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Send sends a Modbus TCP PDU over the connection.
func (t *TCPTransporter) Send(transactionID uint16, unitID uint8, pdu []byte) error {
	frame, errPack := t.packager.Pack(transactionID, unitID, pdu)
	if errPack != nil {
		return errPack
	}
	// Set timeout for write operation
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return err
	}
	defer t.conn.SetDeadline(time.Time{}) // Reset deadline after write
	_, errWrite := t.conn.Write(frame)
	return errWrite
}

// Receive receives a Modbus TCP response from the connection.
func (t *TCPTransporter) Receive() (transactionID uint16, unitID uint8, pdu []byte, err error) {
	// Always reset the deadline once, covering the whole receive operation
	deadline := time.Now().Add(t.timeout)
	if err := t.conn.SetDeadline(deadline); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to set deadline: %w", err)
	}
	defer t.conn.SetDeadline(time.Time{}) // Reset deadline after read
	// Read MBAP Header (7 bytes)
	header := make([]byte, 7)
	if _, err := io.ReadFull(t.conn, header); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to read MBAP header: %w", err)
	}

	transactionID = binary.BigEndian.Uint16(header[0:2])
	protocolID := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	unitID = header[6]

	if protocolID != ProtocolIdentifierTCP {
		return 0, 0, nil, fmt.Errorf("invalid protocol ID: got %d, expected %d", protocolID, ProtocolIdentifierTCP)
	}
	if length == 0 {
		return 0, 0, nil, fmt.Errorf("invalid length: 0")
	}
	if length > 260 { // Based on Modbus spec: 1 (unit id) + 255 (PDU) + 2 (header)
		return 0, 0, nil, fmt.Errorf("length too large: %d", length)
	}

	// Length includes Unit ID, so PDU is (length - 1)
	pduLength := int(length) - 1
	pdu = make([]byte, pduLength)
	if pduLength > 0 {
		if _, err := io.ReadFull(t.conn, pdu); err != nil {
			return 0, 0, nil, fmt.Errorf("failed to read PDU: %w", err)
		}
	}

	// No need to re-unpack; header+PDU already parsed manually
	return transactionID, unitID, pdu, nil
}

// Close closes the underlying connection.
func (t *TCPTransporter) Close() error {
	return t.conn.Close()
}
